package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"take-out/database"
	"take-out/models"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// HandleRegisterShop 处理商家注册的 HTTP 请求
func HandleRegisterShop(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var shop models.Shop
		if err := json.NewDecoder(r.Body).Decode(&shop); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		// 检查商家名称是否已存在
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM shops WHERE shopname = ?)", shop.ShopName).Scan(&exists)
		if err != nil {
			http.Error(w, "商家名称检查失败", http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, "商家名称已存在，请选择其他名称", http.StatusConflict)
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(shop.ShopPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "密码哈希失败", http.StatusInternalServerError)
			return
		}
		shop.ShopPassword = string(passwordHash)

		shopID, err := database.InsertShop(rp, db, &shop)
		if err != nil {
			http.Error(w, fmt.Sprintf("商家注册失败: %v", err), http.StatusInternalServerError)
			return
		}

		shop.ShopID = int(shopID)
		shop.ShopPassword = "" // 清除密码
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(shop)
	}
}

// HandleLoginShop 商家登录
func HandleLoginShop(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var credentials struct {
			ShopName string `json:"shop_name"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		validatedShop, err := database.ValidateShop(db, credentials.ShopName, credentials.Password)
		if err != nil {
			http.Error(w, fmt.Sprintf("登录失败: %v", err), http.StatusUnauthorized)
			return
		}

		// 生成Token
		claims := jwt.MapClaims{"shop_id": validatedShop.ShopID}
		redisKey := fmt.Sprintf("token:shop:%d", validatedShop.ShopID)
		token, err := GenerateToken(claims, redisKey, rp)
		if err != nil {
			http.Error(w, "生成Token失败", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "登录成功",
			"shop_name": validatedShop.ShopName,
			"shop_id":   validatedShop.ShopID,
			"token":     token,
		})
	}
}

// AuthenticateTokenShop 返回商家认证中间件
func AuthenticateTokenShop(rp *database.RedisPool) func(http.Handler) http.Handler {
	return AuthMiddleware(rp, "shop", "shop_id")
}

// HandleGetShops 获取商家列表，支持分页
func HandleGetShops(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		if pageStr == "" {
			pageStr = "1" //默认第一页
		}

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			http.Error(w, "无效的页码参数", http.StatusBadRequest)
			return
		}

		pageSize := 20
		offset := (page - 1) * pageSize

		shops, err := database.QueryShops(db, offset, pageSize)
		if err != nil {
			http.Error(w, fmt.Sprintf("查询商家失败: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(shops)
	}
}

// HandleShopProducts 查询商家商品
func HandleShopProducts(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shopIDStr := r.URL.Query().Get("shop_id")
		if shopIDStr == "" {
			http.Error(w, "缺少商家ID参数", http.StatusBadRequest)
			return
		}

		shopID, err := strconv.Atoi(shopIDStr)
		if err != nil {
			http.Error(w, "无效的商家ID", http.StatusBadRequest)
			return
		}

		cacheKey := fmt.Sprintf("shop_products_%d", shopID)
		data, err := database.GetFromCache(rp, cacheKey)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(data))
			return
		}

		products, err := database.QueryProductsByShopID(db, shopID)
		if err != nil {
			http.Error(w, fmt.Sprintf("查询商品失败: %v", err), http.StatusInternalServerError)
			return
		}

		jsonData, _ := json.Marshal(products)
		database.SetToCache(rp, cacheKey, string(jsonData), time.Hour)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	}
}

// HandleNearbyShops 查询附近商家
func HandleNearbyShops(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latStr := r.URL.Query().Get("lat")
		lngStr := r.URL.Query().Get("lng")
		if latStr == "" || lngStr == "" {
			http.Error(w, "缺少经纬度参数", http.StatusBadRequest)
			return
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			http.Error(w, "无效的纬度参数", http.StatusBadRequest)
			return
		}
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			http.Error(w, "无效的经度参数", http.StatusBadRequest)
			return
		}

		cachedKey := fmt.Sprintf("nearby_shops_%f_%f", lat, lng)
		data, err := database.GetFromCache(rp, cachedKey)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(data))
			return
		}

		shops, err := database.QueryNearbyShops(db, lat, lng)
		if err != nil {
			http.Error(w, fmt.Sprintf("查询附近商家失败: %v", err), http.StatusInternalServerError)
			return
		}

		jsonData, _ := json.Marshal(shops)
		database.SetToCache(rp, cachedKey, string(jsonData), time.Hour)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shops)
	}
}
