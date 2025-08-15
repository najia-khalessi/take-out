//处理商家的商品管理、接单等接口
package database

import (
	"context"
	"database/sql"
	"fmt"
	"take-out/logging"
	"take-out/models"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

//添加商家
func InsertShop(rp *RedisPool, db *sql.DB, shop *models.Shop) (int64, error) {
	logging.Info("Attempting to insert a new shop", logrus.Fields{"shopName": shop.ShopName})
	// 将商家信息插入 PostgreSQL
	query := "INSERT INTO shops (shopname, shoppassword, shopphone, shopaddress, description) VALUES ($1, $2, $3, $4, $5) RETURNING shopid"
	var shopID int64
	err := db.QueryRow(query, shop.ShopName, shop.ShopPassword, shop.ShopPhone, shop.ShopAddress, shop.Description).Scan(&shopID)
	if err != nil {
		logging.Error("Failed to insert shop into PostgreSQL", logrus.Fields{"error": err})
		return 0, fmt.Errorf("商家信息插入PostgreSQL失败: %v", err)
	}

	// 将商家信息插入 Redis
	ctx := context.Background()
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	err = rdb.HMSet(ctx, fmt.Sprintf("shop:%d", shopID), map[string]interface{}{
		"shop_id":       shopID,
		"shop_name":     shop.ShopName,
		"shop_password": shop.ShopPassword,
		"phone":         shop.ShopPhone,
		"address":       shop.ShopAddress,
		"description":   shop.Description,
	}).Err()
	if err != nil {
		logging.Warn("Failed to insert shop into Redis", logrus.Fields{"error": err})
	}

	logging.Info("Successfully inserted new shop", logrus.Fields{"shopID": shopID})
	return shopID, nil
}

// CheckShopExist 检查商店是否存在
func CheckShopExist(db *sql.DB, shopID int) (bool, error) {
	logging.Info("Checking if shop exists", logrus.Fields{"shopID": shopID})
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM shops WHERE shopid = $1)"
	err := db.QueryRow(query, shopID).Scan(&exists)
	if err != nil {
		logging.Error("Failed to check if shop exists", logrus.Fields{"error": err})
		return false, fmt.Errorf("检查商店是否存在失败: %v", err)
	}
	logging.Info("Shop existence check complete", logrus.Fields{"shopID": shopID, "exists": exists})
	return exists, nil
}

//查询商家，支持分页
func QueryShops(db *sql.DB, offset, limit int) ([]models.Shop, error) {
	logging.Info("Querying shops with pagination", logrus.Fields{"offset": offset, "limit": limit})
	query := `
        SELECT shopid, shopname, shopaddress, shopphone, description
        FROM shops
        ORDER BY shopid
        LIMIT $1 OFFSET $2
    `
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		logging.Error("Failed to query shops", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询商家失败: %v", err)
	}
	defer rows.Close()

	var shops []models.Shop
	for rows.Next() {
		var shop models.Shop
		if err := rows.Scan(&shop.ShopID, &shop.ShopName, &shop.ShopAddress, &shop.ShopPhone, &shop.Description); err != nil {
			logging.Error("Failed to scan shop row", logrus.Fields{"error": err})
			return nil, err
		}
		shops = append(shops, shop)
	}
	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		logging.Error("Error while iterating over shop rows", logrus.Fields{"error": err})
		return nil, fmt.Errorf("遍历商家数据失败: %v", err)
	}
	logging.Info("Successfully queried shops", logrus.Fields{"count": len(shops)})
	return shops, nil
}

//查询附近商家
func QueryNearbyShops(db *sql.DB, longitude, latitude float64) ([]models.Shop, error) {
	logging.Info("Querying nearby shops", logrus.Fields{"longitude": longitude, "latitude": latitude})
	const (
		maxDistance = 10 // 最大距离（公里）
		maxResults  = 20 // 最大返回结果数
	)

	query := `
        SELECT shopid, shopname, shopphone, shopaddress, description, shoplatitude, shoplongitude,
               (point(shoplongitude, shoplatitude) <@> point($1, $2)) * 1.609344 AS distance
        FROM shops
        WHERE (point(shoplongitude, shoplatitude) <@> point($1, $2)) * 1.609344 < $3
        ORDER BY distance
        LIMIT $4
    `
	rows, err := db.Query(query, longitude, latitude, maxDistance, maxResults)
	if err != nil {
		logging.Error("Failed to query nearby shops", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询附近商家失败: %v", err)
	}
	defer rows.Close()

	var shops []models.Shop
	for rows.Next() {
		var shop models.Shop
		var distance float64
		if err := rows.Scan(&shop.ShopID,
			&shop.ShopName,
			&shop.ShopPhone,
			&shop.ShopAddress,
			&shop.Description,
			&shop.ShopLatitude,
			&shop.ShopLongitude,
			&distance); err != nil {
			logging.Error("Failed to scan nearby shop row", logrus.Fields{"error": err})
			return nil, fmt.Errorf("解析商家数据失败: %v", err)
		}
		shops = append(shops, shop)
	}
	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		logging.Error("Error while iterating over nearby shop rows", logrus.Fields{"error": err})
		return nil, fmt.Errorf("遍历商家数据失败: %v", err)
	}
	logging.Info("Successfully queried nearby shops", logrus.Fields{"count": len(shops)})
	return shops, nil
}

//查询商家的商品列表
func QueryProductsByShopID(db *sql.DB, shopID int) ([]models.Product, error) {
	logging.Info("Querying products by shop ID", logrus.Fields{"shopID": shopID})
	query := "SELECT productid, productname, description, price, stock FROM products WHERE shopid = $1 ORDER BY productid"
	rows, err := db.Query(query, shopID)
	if err != nil {
		logging.Error("Failed to query products by shop ID", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询商品失败: %v", err)
	}
	defer rows.Close() //将连接还给连接池，不是关闭而是等待复用

	var products []models.Product
	for rows.Next() { //遍历每一行
		var product models.Product
		if err := rows.Scan(&product.ProductID, &product.ProductName, &product.Description, &product.Price, &product.Stock); err != nil {
			logging.Error("Failed to scan product row", logrus.Fields{"error": err})
			return nil, err
		}
		products = append(products, product)
	}
	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		logging.Error("Error while iterating over product rows", logrus.Fields{"error": err})
		return nil, fmt.Errorf("遍历商品数据失败: %v", err)
	}
	logging.Info("Successfully queried products by shop ID", logrus.Fields{"shopID": shopID, "count": len(products)})
	return products, nil
}

// ValidateShop 验证商家凭据
func ValidateShop(db *sql.DB, shopName, password string) (*models.Shop, error) {
	logging.Info("Validating shop credentials", logrus.Fields{"shopName": shopName})
	var shop models.Shop
	query := "SELECT shopid, shopname, shoppassword, shopphone, shopaddress, description FROM shops WHERE shopname = $1"
	err := db.QueryRow(query, shopName).Scan(&shop.ShopID, &shop.ShopName, &shop.ShopPassword, &shop.ShopPhone, &shop.ShopAddress, &shop.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Warn("Shop not found", logrus.Fields{"shopName": shopName})
			return nil, fmt.Errorf("商家不存在")
		}
		logging.Error("Failed to query shop", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询商家失败: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(shop.ShopPassword), []byte(password))
	if err != nil {
		logging.Warn("Incorrect password for shop", logrus.Fields{"shopName": shopName})
		return nil, fmt.Errorf("密码错误")
	}
	logging.Info("Successfully validated shop", logrus.Fields{"shopID": shop.ShopID})
	return &shop, nil
}
