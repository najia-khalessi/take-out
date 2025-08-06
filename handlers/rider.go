package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"take-out/database"
	"take-out/models"

	"github.com/golang-jwt/jwt"
)

// HandleApplyForRider 骑手身份申请
func HandleApplyForRider(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			UserID      int     `json:"user_id"`
			VehicleType string  `json:"vehicle_type"`
			Status      string  `json:"status"`
			Rating      float64 `json:"rating"`
			DeliveryFee float64 `json:"delivery_fee"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		// 验证用户是否存在
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE userid = ?)", request.UserID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "用户不存在", http.StatusBadRequest)
			return
		}

		rider := models.Rider{
			User: models.User{
				UserID: request.UserID,
			},
			VehicleType: request.VehicleType,
			RiderRating: request.Rating,
			RiderStatus: request.Status,
			DeliveryFee: request.DeliveryFee,
		}

		riderID, err := database.InsertRider(rp, db, &rider)
		if err != nil {
			http.Error(w, fmt.Sprintf("骑手身份申请失败: %v", err), http.StatusInternalServerError)
			return
		}
		rider.RiderID = int(riderID)

		// 为骑手生成Token
		claims := jwt.MapClaims{"rider_id": rider.RiderID}
		redisKey := fmt.Sprintf("token:rider:%d", rider.RiderID)
		token, err := GenerateToken(claims, redisKey, rp)
		if err != nil {
			http.Error(w, "生成Token失败", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "骑手注册成功",
			"rider_id": rider.RiderID,
			"token":    token,
		})
	}
}

// AuthenticateTokenRider 返回骑手认证中间件
func AuthenticateTokenRider(rp *database.RedisPool) func(http.Handler) http.Handler {
	return AuthMiddleware(rp, "rider", "rider_id")
}