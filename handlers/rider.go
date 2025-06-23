package handlers

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
	"log"
	"os"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/go-sql-driver/mysql"
    "github.com/golang-jwt/jwt"
    "golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
    
    "take-out/models"
    "take-out/database"
)
// 骑手身份申请
func handleApplyForRider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		UserID      int     `json:"user_id"`
		VehicleType string  `json:"vehicle_type"`
		Status      string  `json:"status"`
		Rating      float64 `json:"rating"`
	}

	// 解析请求体
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "请求体解析错误", http.StatusBadRequest)
		return
	}

	// 验证用户是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ?)", request.UserID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "用户不存在", http.StatusBadRequest)
		return
	}

	// 为用户生成 RiderID 并插入骑手数据
	rider := Rider{
		User:        User{UserID: request.UserID},
		VehicleType: request.VehicleType,
		Rating:      request.Rating,
		RiderStatus: request.Status,
	}

	riderID, err := insertRider(rp, db, &rider)
	if err != nil {
		http.Error(w, fmt.Sprintf("骑手身份申请失败: %v", err), http.StatusInternalServerError)
		return
	}
	rider.RiderID = int(riderID)

	// 为骑手生成Token，并存储到Redis
	token, err := generateTokenRider(rp, rider.RiderID)
	if err != nil {
		http.Error(w, "生成Token失败", http.StatusInternalServerError)
		return
	}

	// 返回创建成功信息和Token
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "骑手注册成功",
		"rider_id": rider.RiderID,
		"token":    token,
	})
}
// 生成JWT Token，适用于骑手，并存储到Redis
func generateTokenRider(rp *RedisPool, riderID int) (string, error) {
	// 创建JWT的Claims
	claims := jwt.MapClaims{
		"rider_id": riderID,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 设置Token有效期为24小时
	}

	// 使用HS256签名算法创建JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用全局密钥签名Token
	tokenString, err := token.SignedString([]byte("your_secret_key"))
	if err != nil {
		return "", err
	}

	// 将生成的Token存储到Redis中，设置过期时间为24小时
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	err = rdb.Set(context.Background(), fmt.Sprintf("token:rider:%d", riderID), tokenString, 24*time.Hour).Err()
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// 验证Token的中间件，适用于骑手
func authenticateTokenRider(rp *RedisPool, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 从请求头中获取 Token
        token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
        if token == "" {
            http.Error(w, "未提供骑手身份验证信息/缺少Token", http.StatusUnauthorized)
            return
        }

        rdb := rp.GetClient()
        defer rp.PutClient(rdb)

        // 解析 Token 获取 riderID
        claims := jwt.MapClaims{}

        // 使用 jwt.ParseWithClaims 解析 Token, 通过密钥验证签名有效性
        _, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
            // 验证签名方法
            if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("非预期的签名方法: %v", t.Header["alg"])
            }
            return []byte(secretKey), nil
        })
        if err != nil {
            http.Error(w, "无效的骑手Token", http.StatusUnauthorized)
            return
        }

        // 验证Token声明
        if err := claims.Valid(); err != nil {
            http.Error(w, "骑手Token声明无效", http.StatusUnauthorized)
            return
        }

        // 获取骑手ID
        riderID := int(claims["rider_id"].(float64))

        // 构造 Redis Key并验证
        ctx := context.Background()
        redisKey := fmt.Sprintf("token:rider:%d", riderID)
        cachedToken, err := rdb.Get(ctx, redisKey).Result()
        if err != nil || cachedToken != token {
            http.Error(w, "骑手Token无效或已过期", http.StatusUnauthorized)
            return
        }

        // 将骑手ID添加到请求上下文
        ctx = context.WithValue(r.Context(), "rider_id", riderID)
        r = r.WithContext(ctx)

        // 验证通过，继续处理请求
        next.ServeHTTP(w, r)
    }
}


