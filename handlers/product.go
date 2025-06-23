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
//HTTP处理函数：添加商品
func HandleUpdateProductStock(w http.ResponseWriter, r *http.Request) {
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
	ProductID, err := addProductForShop(rp, db, product.ShopID, &product)
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