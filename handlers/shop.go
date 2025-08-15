package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"take-out/database"
	"take-out/response"
)

// HandleGetShops 获取商家列表，支持分页
func HandleGetShops(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		if pageStr == "" {
			pageStr = "1" //默认第一页
		}

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			response.ValidationError(w, "页码参数格式错误", "page")
			return
		}

		if page <= 0 {
			page = 1
		}

		pageSize := 20
		offset := (page - 1) * pageSize

		shops, err := database.QueryShops(db, offset, pageSize)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		response.Success(w, map[string]interface{}{
			"list":  shops,
			"total": len(shops),
			"page":  page,
			"size":  pageSize,
		}, "获取商家列表成功")
	}
}

// HandleShopProducts 查询商家商品
func HandleShopProducts(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shopIDStr := r.URL.Query().Get("shop_id")
		if shopIDStr == "" {
			response.ValidationError(w, "商家ID不能为空", "shop_id")
			return
		}

		shopID, err := strconv.Atoi(shopIDStr)
		if err != nil {
			response.ValidationError(w, "商家ID格式错误", "shop_id")
			return
		}

		cacheKey := fmt.Sprintf("shop_products_%d", shopID)
		data, err := database.GetFromCache(rp, cacheKey)
		if err == nil {
			var products []interface{}
			if err := json.Unmarshal([]byte(data), &products); err == nil {
				response.Success(w, products, "获取商家商品成功")
				return
			}
		}

		products, err := database.QueryProductsByShopID(db, shopID)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		jsonData, _ := json.Marshal(products)
		database.SetToCache(rp, cacheKey, string(jsonData), time.Hour)

		response.Success(w, products, "获取商家商品成功")
	}
}

// HandleNearbyShops 查询附近商家
func HandleNearbyShops(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latStr := r.URL.Query().Get("lat")
		lngStr := r.URL.Query().Get("lng")
		if latStr == "" || lngStr == "" {
			response.ValidationError(w, "经纬度参数不能为空", "lat,lng")
			return
		}

		_, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			response.ValidationError(w, "纬度参数格式错误", "lat")
			return
		}

		_, err = strconv.ParseFloat(lngStr, 64)
		if err != nil {
			response.ValidationError(w, "经度参数格式错误", "lng")
			return
		}

		// 这里可以添加距离计算逻辑
		// 暂时返回所有商家
		shops, err := database.QueryShops(db, 0, 100)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		response.Success(w, shops, "获取附近商家成功")
	}
}
