package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"take-out/database"
	"take-out/models"
	"take-out/response"
)

//HTTP处理函数：添加商品
func HandleAddProduct(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//验证 HTTP 方法
		if r.Method != http.MethodPost {
			response.Error(w, "方法未被允许", http.StatusMethodNotAllowed)
			return
		}

		// 从上下文中获取 shopID
		shopID, ok := r.Context().Value("shopID").(int)
		if !ok || shopID == 0 {
			response.Unauthorized(w, "无效的商店ID或权限不足")
			return
		}

		//解析请求体中的 JSON 数据
		var product models.Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		// 参数验证
		if product.ProductName == "" {
			response.ValidationError(w, "商品名称不能为空", "product_name")
			return
		}
		if product.Price <= 0 {
			response.ValidationError(w, "商品价格必须大于0", "price")
			return
		}
		if product.Stock < 0 {
			response.ValidationError(w, "商品库存不能为负数", "stock")
			return
		}

		// 将从上下文中获取的 shopID 赋值给 product
		product.ShopID = shopID

		//添加商品
		ProductID, err := database.AddProductForShop(rp, db, product.ShopID, &product)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		//返回新商品的ID和信息
		product.ProductID = ProductID
		response.Created(w, map[string]int64{"product_id": ProductID}, "商品添加成功")
	}
}

//HTTP处理函数：更新商品库存
func HandleUpdateProductStock(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//验证 HTTP 方法
		if r.Method != http.MethodPost {
			response.Error(w, "方法未被允许", http.StatusMethodNotAllowed)
			return
		}

		//解析请求体中的 JSON 数据
		var product struct {
			ProductID int `json:"product_id"`
			Stock     int `json:"stock"`
		}
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.BadRequest(w, "请求格式错误", "无效的JSON格式")
			return
		}

		//参数验证
		if product.ProductID == 0 {
			response.ValidationError(w, "商品ID不能为空", "product_id")
			return
		}
		if product.Stock < 0 {
			response.ValidationError(w, "库存不能为负数", "stock")
			return
		}

		//更新商品库存
		err := database.UpdateProductStock(rp, db, product.ProductID, product.Stock)
		if err != nil {
			response.ServerError(w, err)
			return
		}

		//返回成功信息
		response.Success(w, nil, "库存更新成功")
	}
}