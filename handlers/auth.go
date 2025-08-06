package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"take-out/database"

	"github.com/golang-jwt/jwt"
)

var jwtSecretKey = []byte(os.Getenv("JWT_SECRET_KEY"))

// GenerateToken 生成适用于任何角色的JWT Token
func GenerateToken(claims jwt.MapClaims, redisKey string, rp *database.RedisPool) (string, error) {
	if string(jwtSecretKey) == "" {
		jwtSecretKey = []byte("your_secret_key") // 仅用于开发环境的后备密钥
	}

	// 设置过期时间
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	claims["iat"] = time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", err
	}

	// 将Token存储到Redis
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	err = rdb.Set(context.Background(), redisKey, tokenString, 24*time.Hour).Err()
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// AuthMiddleware 创建一个通用的认证中间件
func AuthMiddleware(rp *database.RedisPool, role string, idClaimKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if tokenString == "" {
				http.Error(w, "缺少Token", http.StatusUnauthorized)
				return
			}

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("非预期的签名方法: %v", t.Header["alg"])
				}
				return jwtSecretKey, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "无效的Token", http.StatusUnauthorized)
				return
			}

			// 从Token中获取ID
			idFloat, ok := claims[idClaimKey].(float64)
			if !ok {
				http.Error(w, "Token中缺少ID声明", http.StatusUnauthorized)
				return
			}
			id := int(idFloat)

			// 在Redis中验证Token
			redisKey := fmt.Sprintf("token:%s:%d", role, id)
			rdb := rp.GetClient()
			defer rp.PutClient(rdb)
			cachedToken, err := rdb.Get(context.Background(), redisKey).Result()
			if err != nil || cachedToken != tokenString {
				http.Error(w, "Token无效或已过期", http.StatusUnauthorized)
				return
			}

			// 将ID添加到请求上下文中
			ctx := context.WithValue(r.Context(), "id", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
