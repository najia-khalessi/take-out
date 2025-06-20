//处理商家的商品管理、接单等接口

//添加商品，修改，删除，更新库存，

package database

import (
	"context"
    "database/sql"
    "fmt"

    "take-out/models"
    "take-out/database"
)
//添加商家
func insertShop(rp *RedisPool, db *sql.DB, shop *models.Shop) (int64, error) {
    // 将商家信息插入 MySQL
    query := "INSERT INTO shops (shop_name, shop_password, phone, address, description) VALUES (?, ?, ?, ?, ?)"
    result, err := db.Exec(query, shop.ShopName, shop.ShopPassword, shop.Phone, shop.Address, shop.Description)
    if err != nil {
        return 0, fmt.Errorf("商家信息插入MySQL失败: %v", err)
    }
    shopID, err := result.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("获取商家ID失败: %v", err)
    }

    // 将商家信息插入 Redis
    ctx := context.Background()
    rdb := rp.GetClient()
    defer rp.PutClient(rdb)
    err = rdb.HMSet(ctx, fmt.Sprintf("shop:%d", shopID), map[string]interface{}{
        "shop_id":       shopID,
        "shop_name":     shop.ShopName,
        "shop_password": shop.ShopPassword,
        "phone":         shop.Phone,
        "address":       shop.Address,
        "description":   shop.Description,
    }).Err()
    if err != nil {
        return 0, fmt.Errorf("商家信息插入Redis失败: %v", err)
    }

    return shopID, nil
}
//查询商家，支持分页
func QueryShops(db *sql.DB, offset, limit int) ([]models.Shop, error) {
	query := `
        SELECT shop_id, shop_name, address, phone, description
        FROM shops
        ORDER BY shopid
        LIMIT ? OFFSET ?
    `
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询商家失败: %v", err)
	}
	defer rows.Close()

	var shops []models.Shop
	for rows.Next() {
		var shop models.Shop
		if err := rows.Scan(&shop.ShopID, &shop.ShopName, &shop.Address, &shop.Phone, &shop.Description); err != nil {
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
        maxDistance = 10  // 最大距离（公里）
        maxResults  = 20  // 最大返回结果数
    )

    query := `
        SELECT shopid, shopname, shopphone, shopaddress, shopdescription, shoplatitude, shoplongitude,
               ( 6371 * acos( cos( radians(?) ) * cos( radians(latitude) ) *
               cos( radians(longitude) - radians(?) ) + sin( radians(?) ) *
               sin( radians(latitude) ) ) ) AS distance
        FROM shops
        HAVING distance < ?
        ORDER BY distance
        LIMIT ?
    `
	rows, err := db.Query(query, latitude, longitude, latitude,  maxDistance, maxResults)
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
            &shop.Phone,
            &shop.Address,
            &shop.Description,
            &shop.Latitude,
            &shop.Longitude,
            &distance,); err != nil {
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
func QueryProductsByShopID(db *sql.DB, models.ShopID int) ([]modles.Product, error) {
    query := "SELECT productid, productname, prodescription, productprice, stock 
			FROM products 
			WHERE shopid = ?
			ORDER BY productid"
	rows, err := db.Query(query, models.ShopID)
	if err != nil {
		return nil, fmt.Errorf("查询商品失败: %v", err)
	}
	defer rows.Close()   //将连接还给连接池，不是关闭而是等待复用

	var products []Product
	for rows.Next() {    //遍历每一行
		var product Product
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

//接单
// 商家服务启动时调用
    func shopSubscribe(shopID int) {
        rdb := rp.GetClient()
        shopChannel := fmt.Sprintf("shop_%d", shopID)
        
        pubsub := rdb.Subscribe(context.Background(), shopChannel)
        for msg := range pubsub.Channel() {
            var notif map[string]interface{}
            json.Unmarshal([]byte(msg.Payload), &notif)
            switch notif["type"] {
            case "new_order":
                // 触发商家界面弹窗或声音提示
                showOrderNotification(notif["order_id"].(int))
            }
        }
    }