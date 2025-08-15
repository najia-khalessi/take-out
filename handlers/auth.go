package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"take-out/database"
	"take-out/models"
	"take-out/response"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecretKey = []byte(os.Getenv("JWT_SECRET_KEY"))

// GenerateTokenPair generates access and refresh tokens for a user
func GenerateTokenPair(userID int) (models.TokenPair, error) {
	if string(jwtSecretKey) == "" {
		jwtSecretKey = []byte("your_secret_key") // Fallback for dev
	}

	// Create Access Token
	accessTokenClaims := models.Claims{
		UserID: userID,
		Type:   "access",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(jwtSecretKey)
	if err != nil {
		return models.TokenPair{}, err
	}

	// Create Refresh Token
	refreshTokenClaims := models.Claims{
		UserID: userID,
		Type:   "refresh",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
			IssuedAt:  time.Now().Unix(),
			Id:        uuid.New().String(),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecretKey)
	if err != nil {
		return models.TokenPair{}, err
	}

	return models.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// GenerateShopTokenPair generates access and refresh tokens for a shop
func GenerateShopTokenPair(shopID int) (models.TokenPair, error) {
	if string(jwtSecretKey) == "" {
		jwtSecretKey = []byte("your_secret_key") // Fallback for dev
	}

	// Create Access Token
	accessTokenClaims := models.Claims{
		ShopID: shopID,
		Type:   "access",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(jwtSecretKey)
	if err != nil {
		return models.TokenPair{}, err
	}

	// Create Refresh Token
	refreshTokenClaims := models.Claims{
		ShopID: shopID, 
		Type:   "refresh",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
			IssuedAt:  time.Now().Unix(),
			Id:        uuid.New().String(),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecretKey)
	if err != nil {
		return models.TokenPair{}, err
	}

	return models.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// RefreshToken handles token renewal
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "请求格式错误", "无效的JSON格式")
		return
	}

	claims, err := ParseToken(body.RefreshToken)
	if err != nil || claims.Type != "refresh" {
		response.Unauthorized(w, "无效的刷新令牌")
		return
	}

	// Check blacklist
	isBlacklisted, err := database.IsTokenBlacklisted(claims.Id)
	if err != nil {
		response.ServerError(w, err)
		return
	}
	if isBlacklisted {
		response.Unauthorized(w, "刷新令牌已被吊销")
		return
	}

	// Blacklist the old refresh token
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if err := database.AddTokenToBlacklist(claims.Id, expiresAt); err != nil {
		response.ServerError(w, err)
		return
	}

	// Generate new token pair
	var tokenPair models.TokenPair
	if claims.UserID != 0 {
		tokenPair, err = GenerateTokenPair(claims.UserID)
	} else if claims.ShopID != 0 {
		tokenPair, err = GenerateShopTokenPair(claims.ShopID)
	} else if claims.RiderID != 0 {
		tokenPair, err = GenerateRiderTokenPair(claims.RiderID)
	}
	
	if err != nil {
		response.ServerError(w, err)
		return
	}

	response.Success(w, tokenPair, "令牌刷新成功")
}

// Logout invalidates a refresh token
func Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "请求格式错误", "无效的JSON格式")
		return
	}

	claims, err := ParseToken(body.RefreshToken)
	if err != nil || claims.Type != "refresh" {
		response.Unauthorized(w, "无效的刷新令牌")
		return
	}

	// Blacklist the refresh token
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if err := database.AddTokenToBlacklist(claims.Id, expiresAt); err != nil {
		response.ServerError(w, err)
		return
	}

	response.Success(w, nil, "登出成功")
}

// ParseToken validates and parses a JWT token string
func ParseToken(tokenString string) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.NewValidationError("invalid token", jwt.ValidationErrorMalformed)
	}

	return claims, nil
}

// HandleUserRegister handles user registration
func HandleUserRegister(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		// 参数验证
		if user.UserPhone == "" || user.UserPassword == "" {
			response.ValidationError(w, "手机号和密码不能为空", "user_phone,user_password")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.UserPassword), bcrypt.DefaultCost)
		if err != nil {
			response.ServerError(w, err)
			return
		}
		user.UserPassword = string(hashedPassword)

		userID, err := database.InsertUser(rp, db, &user)
		if err != nil {
			// 检查是否为重复注册
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already exists") {
				response.ValidationError(w, "该手机号已被注册", "user_phone")
			} else {
				response.ServerError(w, err)
			}
			return
		}
		user.UserID = int(userID)

		user.UserPassword = "" // Do not return password

		response.Created(w, user, "用户注册成功")
	}
}

// HandleShopRegister handles shop registration
func HandleShopRegister(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var shop models.Shop
		if err := json.NewDecoder(r.Body).Decode(&shop); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		// 参数验证
		if shop.ShopPhone == "" || shop.ShopPassword == "" || shop.ShopName == "" {
			response.ValidationError(w, "店铺名称、手机号和密码不能为空", "shop_name,shop_phone,shop_password")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(shop.ShopPassword), bcrypt.DefaultCost)
		if err != nil {
			response.ServerError(w, err)
			return
		}
		shop.ShopPassword = string(hashedPassword)

		shopID, err := database.InsertShop(rp, db, &shop)
		if err != nil {
			// 检查是否为重复注册
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already exists") {
				response.ValidationError(w, "该手机号已被注册", "shop_phone")
			} else {
				response.ServerError(w, err)
			}
			return
		}
		shop.ShopID = int(shopID)

		shop.ShopPassword = "" // Do not return password

		response.Created(w, shop, "店铺注册成功")
	}
}

// HandleRiderRegister handles rider registration
func HandleRiderRegister(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rider models.Rider
		if err := json.NewDecoder(r.Body).Decode(&rider); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		// 参数验证
		if rider.RiderPhone == "" || rider.RiderPassword == "" {
			response.ValidationError(w, "手机号和密码不能为空", "rider_phone,rider_password")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rider.RiderPassword), bcrypt.DefaultCost)
		if err != nil {
			response.ServerError(w, err)
			return
		}
		rider.RiderPassword = string(hashedPassword)

		riderID, err := database.InsertRider(db, &rider)
		if err != nil {
			// 检查是否为重复注册
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already exists") {
				response.ValidationError(w, "该手机号已被注册", "rider_phone")
			} else {
				response.ServerError(w, err)
			}
			return
		}
		rider.RiderID = int(riderID)

		rider.RiderPassword = "" // Do not return password

		response.Created(w, rider, "骑手注册成功")
	}
}

// HandleUserLogin handles user login
func HandleUserLogin(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			UserPhone    string `json:"user_phone"`
			UserPassword string `json:"user_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		// 参数验证
		if creds.UserPhone == "" || creds.UserPassword == "" {
			response.ValidationError(w, "手机号和密码不能为空", "user_phone,user_password")
			return
		}

		user, err := database.ValidateUser(db, creds.UserPhone, creds.UserPassword)
		if err != nil {
			response.Unauthorized(w, "账号或密码错误")
			return
		}

		tokenPair, err := GenerateTokenPair(user.UserID)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		response.Success(w, tokenPair, "登录成功")
	}
}

// GenerateRiderTokenPair generates access and refresh tokens for a rider
func GenerateRiderTokenPair(riderID int) (models.TokenPair, error) {
	if string(jwtSecretKey) == "" {
		jwtSecretKey = []byte("your_secret_key") // Fallback for dev
	}

	// Create Access Token
	accessTokenClaims := models.Claims{
		RiderID: riderID, 
		Type:   "access",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(jwtSecretKey)
	if err != nil {
		return models.TokenPair{}, err
	}

	// Create Refresh Token
	refreshTokenClaims := models.Claims{
		RiderID: riderID,
		Type:   "refresh",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
			IssuedAt:  time.Now().Unix(),
			Id:        uuid.New().String(),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecretKey)
	if err != nil {
		return models.TokenPair{}, err
	}

	return models.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// HandleRiderLogin handles rider login
func HandleRiderLogin(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			RiderPhone    string `json:"rider_phone"`
			RiderPassword string `json:"rider_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		// 参数验证
		if creds.RiderPhone == "" || creds.RiderPassword == "" {
			response.ValidationError(w, "手机号和密码不能为空", "rider_phone,rider_password")
			return
		}

		rider, err := database.ValidateRider(db, creds.RiderPhone, creds.RiderPassword)
		if err != nil {
			response.Unauthorized(w, "账号或密码错误")
			return
		}

		tokenPair, err := GenerateRiderTokenPair(rider.RiderID)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		response.Success(w, tokenPair, "登录成功")
	}
}

// HandleShopLogin handles shop login
func HandleShopLogin(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			ShopPhone    string `json:"shop_phone"`
			ShopPassword string `json:"shop_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		// 参数验证
		if creds.ShopPhone == "" || creds.ShopPassword == "" {
			response.ValidationError(w, "手机号和密码不能为空", "shop_phone,shop_password")
			return
		}

		shop, err := database.ValidateShop(db, creds.ShopPhone, creds.ShopPassword)
		if err != nil {
			response.Unauthorized(w, "账号或密码错误")
			return
		}

		tokenPair, err := GenerateShopTokenPair(shop.ShopID)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		response.Success(w, tokenPair, "登录成功")
	}
}

// AuthenticateToken 返回用户认证中间件
func AuthenticateToken(rp *database.RedisPool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "缺少授权头")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ParseToken(tokenString)
			if err != nil {
				response.Unauthorized(w, "无效令牌")
				return
			}

			if claims.Type != "access" || claims.UserID == 0 {
				response.Unauthorized(w, "无效的用户令牌")
				return
			}

			// Add user_id to context
			ctx := context.WithValue(r.Context(), "userID", claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthenticateTokenRider 返回骑手认证中间件
func AuthenticateTokenRider(rp *database.RedisPool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "缺少授权头")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ParseToken(tokenString)
			if err != nil {
				response.Unauthorized(w, "无效令牌")
				return
			}

			if claims.Type != "access" || claims.RiderID == 0 {
				response.Unauthorized(w, "无效的骑手令牌")
				return
			}

			// Add rider_id to context
			ctx := context.WithValue(r.Context(), "riderID", claims.RiderID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HandleRefreshToken 包装RefreshToken函数以符合http.HandlerFunc接口
func HandleRefreshToken(rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RefreshToken(w, r)
	}
}

// AuthenticateTokenShop returns shop authentication middleware
func AuthenticateTokenShop(rp *database.RedisPool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "缺少授权头")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ParseToken(tokenString)
			if err != nil {
				response.Unauthorized(w, "无效令牌")
				return
			}

			if claims.Type != "access" || claims.ShopID == 0 {
				response.Unauthorized(w, "无效的店铺令牌")
				return
			}

			// Add shop_id to context
			ctx := context.WithValue(r.Context(), "shopID", claims.ShopID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
