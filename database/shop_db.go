//处理商家的商品管理、接单等接口
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"take-out/models"

	"golang.org/x/crypto/bcrypt"
)

//添加商家
func InsertShop(rp *RedisPool, db *sql.DB, shop *models.Shop) (int64, error) {
	// 将商家信息插入 PostgreSQL
	query := "INSERT INTO shops (shopname, shoppassword, shopphone, shopaddress, description) VALUES ($1, $2, $3, $4, $5) RETURNING shopid"
	var shopID int64
	err := db.QueryRow(query, shop.ShopName, shop.ShopPassword, shop.ShopPhone, shop.ShopAddress, shop.Description).Scan(&shopID)
	if err != nil {
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
		log.Printf("警告: 商家信息插入Redis失败: %v", err)
	}

	return shopID, nil
}

// CheckShopExist 检查商店是否存在
func CheckShopExist(db *sql.DB, shopID int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM shops WHERE shopid = $1)"
	err := db.QueryRow(query, shopID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查商店是否存在失败: %v", err)
	}
	return exists, nil
}

//查询商家，支持分页
func QueryShops(db *sql.DB, offset, limit int) ([]models.Shop, error) {
	query := `
        SELECT shopid, shopname, shopaddress, shopphone, description
        FROM shops
        ORDER BY shopid
        LIMIT $1 OFFSET $2
    `
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询商家失败: %v", err)
	}
	defer rows.Close()

	var shops []models.Shop
	for rows.Next() {
		var shop models.Shop
		if err := rows.Scan(&shop.ShopID, &shop.ShopName, &shop.ShopAddress, &shop.ShopPhone, &shop.Description); err != nil {
			return nil, err
		}
		shops = append(shops, shop)
	}
	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商家数据失败: %v", err)
	}
	return shops, nil
}

//查询附近商家
func QueryNearbyShops(db *sql.DB, longitude, latitude float64) ([]models.Shop, error) {

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
			return nil, fmt.Errorf("解析商家数据失败: %v", err)
		}
		shops = append(shops, shop)
	}
	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商家数据失败: %v", err)
	}
	return shops, nil
}

//查询商家的商品列表
func QueryProductsByShopID(db *sql.DB, shopID int) ([]models.Product, error) {
	query := "SELECT productid, productname, description, price, stock FROM products WHERE shopid = $1 ORDER BY productid"
	rows, err := db.Query(query, shopID)
	if err != nil {
		return nil, fmt.Errorf("查询商品失败: %v", err)
	}
	defer rows.Close() //将连接还给连接池，不是关闭而是等待复用

	var products []models.Product
	for rows.Next() { //遍历每一行
		var product models.Product
		if err := rows.Scan(&product.ProductID, &product.ProductName, &product.Description, &product.Price, &product.Stock); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商品数据失败: %v", err)
	}
	return products, nil
}

// ValidateShop 验证商家凭据
func ValidateShop(db *sql.DB, shopName, password string) (*models.Shop, error) {
	var shop models.Shop
	query := "SELECT shopid, shopname, shoppassword, shopphone, shopaddress, description FROM shops WHERE shopname = $1"
	err := db.QueryRow(query, shopName).Scan(&shop.ShopID, &shop.ShopName, &shop.ShopPassword, &shop.ShopPhone, &shop.ShopAddress, &shop.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("商家不存在")
		}
		return nil, fmt.Errorf("查询商家失败: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(shop.ShopPassword), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("密码错误")
	}
	return &shop, nil
}
