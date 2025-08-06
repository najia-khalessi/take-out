package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"take-out/database"
	"take-out/models"
)

//HTTP处理函数：添加商品
func HandleAddProduct(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//验证 HTTP 方法
		if r.Method != http.MethodPost {
			http.Error(w, "方法未被允许", http.StatusMethodNotAllowed)
			return
		}

		//解析请求体中的 JSON 数据
		var product models.Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			http.Error(w, "解析请求体失败", http.StatusBadRequest)
			return
		}

		//确保shopID是有效的
		if product.ShopID == 0 {
			http.Error(w, "无效的商店ID", http.StatusBadRequest)
			return
		}
		//添加商品
		ProductID, err := database.AddProductForShop(rp, db, product.ShopID, &product)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//返回新商品的ID
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int64{
			"productid": ProductID,
		})
	}
}

//HTTP处理函数：更新商品库存
func HandleUpdateProductStock(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//验证 HTTP 方法
		if r.Method != http.MethodPost {
			http.Error(w, "方法未被允许", http.StatusMethodNotAllowed)
			return
		}

		//解析请求体中的 JSON 数据
		var product models.Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			http.Error(w, "解析请求体失败", http.StatusBadRequest)
			return
		}

		//确保productID是有效的
		if product.ProductID == 0 {
			http.Error(w, "无效的商品ID", http.StatusBadRequest)
			return
		}
		//更新商品库存
		err := database.UpdateProductStock(rp, db, product.ProductID, product.Stock)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//返回成功信息
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
		})
	}
}