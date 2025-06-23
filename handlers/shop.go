package handlers

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/go-sql-driver/mysql"
    "github.com/golang-jwt/jwt"
    "golang.org/x/crypto/bcrypt"
    
    "take-out/models"
    "take-out/database"
)
//商家注册：handleRegisterShop 处理商家注册的 HTTP 请求
func handleRegisterShop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	var shop Shop
	if err := json.NewDecoder(r.Body).Decode(&shop); err != nil {
		http.Error(w, "请求体解析错误", http.StatusBadRequest)
		return
	}

	// 检查商家名称是否已存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM shops WHERE shop_name = ?)", shop.ShopName).Scan(&exists)
	if err != nil {
		http.Error(w, "商家名称检查失败", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "商家名称已存在，请选择其他名称", http.StatusConflict)
		return
	}

	// 哈希密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(shop.ShopPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "密码哈希失败", http.StatusInternalServerError)
		return
	}
	shop.ShopPassword = string(passwordHash)

	// 插入商家数据，将商家信息存入数据库，
	shopID, err := insertShop(rp, db, &shop)
	if err != nil {
		http.Error(w, fmt.Sprintf("商家注册失败: %v", err), http.StatusInternalServerError)
		return
	}

	shop.ShopID = int(shopID)  //保存新创建的商家ID
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shop)
}
// 商家登录
func handleLoginShop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	var credentials struct {
		ShopName string `json:"shop_name"`
		Password string `json:"password"`
	}

	// 解析请求体
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "请求体解析错误", http.StatusBadRequest)
		return
	}

	// 使用 ValidateShop 函数检查商家凭据
	validatedShop, err := ValidateShop(db, credentials.ShopName, credentials.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf("登录失败: %v", err), http.StatusUnauthorized)
		return
	}

	// 生成Token，并存储到Redis
	token, err := generateTokenShop(rp, validatedShop.ShopID)
	if err != nil {
		http.Error(w, "生成Token失败", http.StatusInternalServerError)
		return
	}

	// 返回登录成功信息和Token
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "登录成功",
		"shop_name": validatedShop.ShopName,
		"shop_id":   validatedShop.ShopID,
		"token":     token,
	})
}
//验证商家凭据
func ValidateShop(db *sql.DB, shopName, password string) (*Shop, error) {
	var shop Shop
	err := db.QueryRow("SELECT shop_id, shop_name, shop_password FROM shops WHERE shop_name = ?",shopName)
	.Scan(&shop.ShopID, &shop.ShopName, &shop.ShopPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("商家不存在")
		}
		return nil, fmt.Errorf("查询商家信息失败: %v", err)
	}

	// 比较哈希后的密码和输入的密码
	if err := bcrypt.CompareHashAndPassword([]byte(shop.ShopPassword), []byte(password)); err != nil {
		return nil, fmt.Errorf("密码错误")
	}

	return &shop, nil
}

// 生成JWT Token，并存储到Redis
func generateTokenShop(rp *RedisPool, shopID int) (string, error) {
	// 创建JWT的Claims
	claims := jwt.MapClaims{
		"shop_id": shopID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 设置Token有效期为24小时
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

	err = rdb.Set(context.Background(), fmt.Sprintf("token:shop:%d", shopID), tokenString, 24*time.Hour).Err()
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// 验证Token的中间件，适用于商家
func authenticateTokenShop(rp *RedisPool, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 从请求头中获取 Token
        token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
        if token == "" {
            http.Error(w, "未提供商家身份验证信息/缺少Token", http.StatusUnauthorized)
            return
        }

        rdb := rp.GetClient()
        defer rp.PutClient(rdb)

        // 解析 Token 获取 shopID
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
            http.Error(w, "无效的商家Token", http.StatusUnauthorized)
            return
        }

        // 验证Token声明
        if err := claims.Valid(); err != nil {
            http.Error(w, "商家Token声明无效", http.StatusUnauthorized)
            return
        }

        // 获取商家ID
        shopID := int(claims["shop_id"].(float64))

        // 构造 Redis Key并验证
        ctx := context.Background()
        redisKey := fmt.Sprintf("token:shop:%d", shopID)
        cachedToken, err := rdb.Get(ctx, redisKey).Result()
        if err != nil || cachedToken != token {
            http.Error(w, "商家Token无效或已过期", http.StatusUnauthorized)
            return
        }

        // 将商家ID添加到请求上下文
        ctx = context.WithValue(r.Context(), "shop_id", shopID)
        r = r.WithContext(ctx)

        // 验证通过，继续处理请求
        next.ServeHTTP(w, r)
    }
}


// 获取商家列表，支持分页
//实现了一个分页的商家列表查询API，支持通过page参数控制页码，每页固定返回20条商家信息，并以JSON格式返回给客户端
func handleGetShops(w http.ResponseWriter, r *http.Request) {
    //获取查询参数，每页20位商家
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		pageStr = "1"  //默认第一页
	}

	//将页码转换为整数：将字符串类型的页码转换为整数类型
	page, err := strconv.Atoi(pageStr)   //strconv.Atoi()：字符串转整数函数
	if err != nil {
		http.Error(w, "无效的页码参数", http.StatusBadRequest)  //http.StatusBadRequest = 400状态码
		return
	}
	
	// 每页显示 20 条商家
	pageSize := 20
	offset := (page - 1) * pageSize  //跳过的记录数
	
	// 查询商家列表，传递 offset 和 limit
	shops, err := QueryShops(db, offset, pageSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("查询商家失败: %v", err), http.StatusInternalServerError)  //返回500内部服务器错误
		return
	}

	// 返回商家列表： 跨语言通信需要统一格式，这里使用JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shops)
}


// 查询商家商品
func handleShopProducts(w http.ResponseWriter, r *http.Request) {
    // 获取商家ID参数
    shopIDStr := r.URL.Query().Get("shop_id")
    if shopIDStr == "" {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "缺少商家ID参数",
        })
        return
    }
    
    shopID, err := strconv.Atoi(shopIDStr)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "无效的商家ID",
        })
        return
    }

    // 构建缓存key
    cacheKey := fmt.Sprintf("shop_products_%d", shopID)
    
    // 尝试从缓存获取
    data, err := GetFromCache(rp, cacheKey)

    if err == nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(data))
        return
    }

    // 缓存未命中，查询数据库
    products, err := QueryProductsByShopID(db, shopID)
	// 缓存命中处理
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{
            "error": fmt.Sprintf("查询商品失败: %v", err),
        })
        return
    }

    // 写入缓存
    jsonData, _ := json.Marshal(products)
    SetToCache(rp, cacheKey, string(jsonData), time.Hour)
    
    // 返回数据
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(products)
}


//查询附近商家
func handleNearbyShops(w http.ResponseWriter, r *http.Request) {
	//从 URL 获取参数
    latStr := r.URL.Query().Get("lat")
    lngStr := r.URL.Query().Get("lng")
	if latStr == "" || lngStr == "" {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{   //统一返回Json格式，方便前端处理
			"error": "缺少经纬度参数",
		})
		return
	}
	//参数类型转换
    lat, err := strconv.ParseFloat(latStr, 64)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "无效的纬度参数",
        })
        return
    }
    lng, err := strconv.ParseFloat(lngStr, 64)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "无效的经度参数",
        })
        return
}
	//使用经纬度构建唯一的缓存键
	cachedKey := fmt.Sprintf("nearby_shops_%f_%f", lat, lng)

	//从 Redis 中获取附近商家数据
	data, err := GetFromCache(rp, cachedKey)    //从 Redis 获取的date是json格式的string
	// 缓存命中处理
	if err == nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(data))
		return
	}

	//查询数据库
	shops, err := QueryNearbyShops(db, lat, lng)  //返回的 `shops` 是结构体切片类型 `[]Shop`
	if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("查询附近商家失败: %v", err),
		})
		return
	}

	//将查询结果转换成json格式后写入redis缓存
	jsonData, _ := json.Marshal(shops)
	SetToCache(rp, cachedKey, string(jsonData), time.Hour)

	//返回查询结果
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(shops)
}

