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
// 用户注册
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}
	//代码检查了请求方法是否为 POST，如果不是，则返回 405 Method Not Allowed 状态码。
	// 因为用户注册通常是一个创建操作，应该使用 POST 方法，通常用于创建资源

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "请求体解析错误", http.StatusBadRequest)  //请求体失败，400 客户端错误
		return
	}

	// 哈希密码：哈希是单向函数，无法反向计算，防止密码破解，保护用户隐私 
	//原始密码转为字节数组
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码哈希失败: %v", err)
		http.Error(w, "密码哈希失败", http.StatusInternalServerError)
		return
	}
	//  存储哈希后的密码（string类型）
	user.Password = string(passwordHash)
	//  在返回给客户端前清除密码
    defer func() {
        user.Password = ""
    }()

	// 插入用户数据
	userID, err := insertUser(rp, db, &user)
	if err != nil {
		log.Printf("用户注册失败: %v", err)
		http.Error(w, fmt.Sprintf("用户注册失败: %v", err), http.StatusInternalServerError)
		return
	}

	user.UserID = int(userID)
	w.WriteHeader(http.StatusCreated)  //201 Created，表示请求成功并且服务器创建了新的资源
	json.NewEncoder(w).Encode(user)
}

//用户注册后怎么维持登录状态？
//用户登录后，服务器会生成一个唯一的token，并将其发送给客户端。客户端在后续的请求中，需要将这个token放在请求头中，以便服务器验证用户的身份。
//用户登录
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	// 使用与 User 结构体对应的登录请求结构
    var loginRequest struct {
        Username string `json:"username"`
        Password string `json:"password"`
        Phone    string `json:"phone,omitempty"`    // 可选字段
        Address  string `json:"address,omitempty"`  // 可选字段
    }
	if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
		http.Error(w, "请求体解析错误", http.StatusBadRequest)
		return
	}

	// 使用 ValidateUser 函数检查用户凭据，如果用户名或密码错误，将返回错误
	validatedUser, err := ValidateUser(db, loginRequest.Username, loginRequest.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf("登录失败: %v", err), http.StatusUnauthorized)
		return
	}
    
	// 生成Token并存储到Redis
	accessToken, err := generateTokenAndStoreInRedis(rp, validatedUser.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("生成短期Token 失败: %v", err), http.StatusInternalServerError)
		return
	}
	// 生成 Refresh Token（随机字符串，非 JWT）
    refreshToken := uuid.New().String() 

	// 存储 Refresh Token 到 Redis（有效期 7 天）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	refreshKey := fmt.Sprintf("refresh:%d", validatedUser.UserID)
	if err := rdb.Set(ctx, refreshKey, refreshToken, 7*24*time.Hour).Err(); err != nil {
		log.Printf("存储 Refresh Token 失败: %v", err)
		http.Error(w, "系统错误", http.StatusInternalServerError)
		return
	}

	// 返回双 Token 给客户端
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken, // 新增短期Token
		"user_id":       validatedUser.UserID,
	})
}
//验证凭证
func ValidateUser(db *sql.DB, username, password string) (*models.User, error) {
    user := &models.User{}
    
    // 从数据库获取哈希后的密码
    err := db.QueryRow(`
        SELECT user_id, username, password 
        FROM users 
        WHERE username = ?
    `, username).Scan(
        &user.UserID,
        &user.Username,
        &user.Password,   // 这里获取的是哈希后的密码字符串
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("用户不存在")
        }
        return nil, fmt.Errorf("数据库查询失败: %v", err)
    }

    // 验证密码
	// - storedHash: 数据库中存储的哈希密码(string转[]byte)
    // - inputPassword: 用户输入的原始密码(string转[]byte)
    if err := bcrypt.CompareHashAndPassword(
        []byte(user.Password), 
        []byte(password),
    ); err != nil {
        return nil, fmt.Errorf("密码错误")
    }

    // 清除密码字段，避免返回给客户端
    user.Password = ""
    
    return user, nil
}
// 生成JWT Token，并存储Token到Redis
func generateTokenAndStoreInRedis(rp *RedisPool, userID int) (string, error) {
    // 设置 JWT 的声明信息
    claims := jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(time.Hour * 24 * 30).Unix(), // 一个月过期
        "iat":     time.Now().Unix(),                          // 令牌签发时间
        "iss":     "take-out-system",                          // 令牌签发者。这两个是声明JWT载荷部分
    }

    // token就是一整个JWT，使用HS256算法创建JWT令牌，令牌格式：Header.Payload.Signature
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)  //声明令牌类型（固定为 JWT）和签名算法HS256是对称加密，在分布式中容易泄露，RS256是非对称加密，安全性更高

    // 从环境变量或配置文件获取密钥
    secretKey := os.Getenv("JWT_SECRET_KEY")
    if secretKey == "" {
        secretKey = "your_secret_key" // 仅用于开发环境，在生产环境中应避免使用默认值
    }

    // 签名生成 token 字符串，使用密钥（secret）和指定算法生成签名
    tokenString, err := token.SignedString([]byte(secretKey))
    if err != nil {
        return "", fmt.Errorf("生成token失败: %v", err)
    }

    // 创建上下文并设置超时
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)  //设置超时时间
    defer cancel()
    rdb := rp.GetClient()
    defer rp.PutClient(rdb)

    // 存储到 Redis，使用 userID 作为 key 的一部分
    redisKey := fmt.Sprintf("token:%d", userID)
    if err := rdb.Set(ctx, redisKey, tokenString, time.Hour*24*30).Err(); err != nil {
        return "", fmt.Errorf("存储token到Redis失败: %v", err)
    }

    return tokenString, nil
}



// 验证 Token 的中间件
/*authenticateToken 是一个函数，它：
1. 接收一个 http.HandlerFunc 作为参数
2. 返回一个新的 http.HandlerFunc
3. 在原有处理函数之前添加 Token 验证逻辑*/
func authenticateToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从请求头中获取 Token
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Error(w, "未提供身份验证信息/缺少Token", http.StatusUnauthorized)
			return
		}

		rdb := rp.GetClient()
		defer rp.PutClient(rdb)

		// 解析 Token 获取 userID
        claims := jwt.MapClaims{}

		//使用 jwt.ParseWithClaims 解析 Token, 通过密钥验证签名有效性
        _, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
            return []byte(secretKey), nil
        })
		if err != nil {
			http.Error(w, "无效的Token", http.StatusUnauthorized)
			return
		}
		// 使用 jwt.ParseWithClaims 解析 Token，并获取声明信息
		if err := claims.Valid(); err != nil {
			http.Error(w, "Token声明无效", http.StatusUnauthorized)
			return
		}	

		userID := int(claims["user_id"].(float64))

		//构造 Redis Key 格式 token:{userID}，比对缓存中的 Token 是否匹配
		ctx := context.Background()
        redisKey := fmt.Sprintf("token:%d", int(userID))
        cachedToken, err := rdb.Get(ctx, redisKey).Result()
		if err != nil || cachedToken != token {
			 http.Error(w, "Token无效或已过期", http.StatusUnauthorized) 
			 return  }

			 //请求放行，所有验证通过后，调用 next.ServeHTTP(w, r) 执行后续业务逻辑
		next.ServeHTTP(w, r)
	}
}
// 刷新 Token
func handleRefreshToken(w http.ResponseWriter, r *http.Request) {
    var req struct{ RefreshToken string `json:"refresh_token"` }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "无效请求", http.StatusBadRequest)
        return
    }

    // 从 Redis 验证 Refresh Token
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    rdb := rp.GetClient()
    defer rp.PutClient(rdb)

    // 遍历匹配 Refresh Token 所属用户（实际生产环境需优化）
    keys, _ := rdb.Keys(ctx, "refresh:*").Result()
    for _, key := range keys {
        storedToken, _ := rdb.Get(ctx, key).Result()
        if storedToken == req.RefreshToken {
            userID, _ := strconv.Atoi(strings.Split(key, ":")[1])
            
            // 生成新 Access Token
            newAccessToken, _ := generateTokenAndStoreInRedis(rp, userID)
            json.NewEncoder(w).Encode(map[string]string{"access_token": newAccessToken})
            return
        }
    }
    http.Error(w, "Refresh Token 无效", http.StatusUnauthorized)
}