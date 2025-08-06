package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"take-out/database"
	"take-out/models"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// HandleRegister 处理用户注册
func HandleRegister(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.UserPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("密码哈希失败: %v", err)
			http.Error(w, "密码哈希失败", http.StatusInternalServerError)
			return
		}
		user.UserPassword = string(passwordHash)

		userID, err := database.InsertUser(rp, db, &user)
		if err != nil {
			log.Printf("用户注册失败: %v", err)
			http.Error(w, fmt.Sprintf("用户注册失败: %v", err), http.StatusInternalServerError)
			return
		}

		user.UserID = int(userID)
		user.UserPassword = "" // 清除密码
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

// HandleLogin 处理用户登录
func HandleLogin(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		validatedUser, err := database.ValidateUser(db, req.Username, req.Password)
		if err != nil {
			http.Error(w, fmt.Sprintf("登录失败: %v", err), http.StatusUnauthorized)
			return
		}

		// 生成 Access Token
		claims := jwt.MapClaims{"user_id": validatedUser.UserID}
		redisKey := fmt.Sprintf("token:user:%d", validatedUser.UserID)
		accessToken, err := GenerateToken(claims, redisKey, rp)
		if err != nil {
			http.Error(w, "生成Token失败", http.StatusInternalServerError)
			return
		}

		// 生成并存储 Refresh Token
		refreshToken := uuid.New().String()
		refreshRedisKey := fmt.Sprintf("refresh:%s", refreshToken)
		rdb := rp.GetClient()
		defer rp.PutClient(rdb)
		err = rdb.Set(context.Background(), refreshRedisKey, validatedUser.UserID, 7*24*time.Hour).Err()
		if err != nil {
			log.Printf("存储 Refresh Token 失败: %v", err)
			http.Error(w, "系统错误", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"user_id":       validatedUser.UserID,
		})
	}
}

// HandleRefreshToken 处理刷新Token的请求
func HandleRefreshToken(rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "无效请求", http.StatusBadRequest)
			return
		}

		// 直接从Redis验证Refresh Token
		rdb := rp.GetClient()
		defer rp.PutClient(rdb)
		refreshRedisKey := fmt.Sprintf("refresh:%s", req.RefreshToken)
		userIDStr, err := rdb.Get(context.Background(), refreshRedisKey).Result()
		if err != nil {
			http.Error(w, "Refresh Token 无效或已过期", http.StatusUnauthorized)
			return
		}

		// 生成新的 Access Token
		claims := jwt.MapClaims{"user_id": userIDStr}
		redisKey := fmt.Sprintf("token:user:%s", userIDStr)
		newAccessToken, err := GenerateToken(claims, redisKey, rp)
		if err != nil {
			http.Error(w, "生成新Token失败", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"access_token": newAccessToken})
	}
}

// AuthenticateToken 返回用户认证中间件
func AuthenticateToken(rp *database.RedisPool) func(http.Handler) http.Handler {
	return AuthMiddleware(rp, "user", "user_id")
}

// protectedEndpoint 是一个受保护的端点示例
func protectedEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("这是一个受保护的端点"))
}