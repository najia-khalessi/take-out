package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"take-out/models"
)

//添加商品到商店
func AddProductForShop(rp *RedisPool, db *sql.DB, ShopID int, product *models.Product) (int64, error) {
	//TODO: 检查商店是否存在
	// shopexist := db.CheckShopExist(ShopID)
	// if shopexist == false {
	//     return 0, fmt.Errorf("商店不存在")
	// }

	//添加商品到商店
	query := `INSERT INTO products (shopid, productname, price, description, stock)
			 VALUES ($1, $2, $3, $4, $5) RETURNING productid`
	var productID int64
	err := db.QueryRow(query, product.ShopID, product.ProductName, product.Price, product.Description, product.Stock).Scan(&productID)
	if err != nil {
		return 0, fmt.Errorf("添加商品失败: %v", err)
	}

	//插入商品到Redis
	rdb := rp.GetClient()          //获取连接
	defer rp.PutClient(rdb)  //确保归还连接

	err = rdb.HSet(context.Background(),
		fmt.Sprintf("product:%d", productID),
		"product_id", productID,
		"shop_id", product.ShopID,
		"product_name", product.ProductName,
		"price", product.Price,
		"description", product.Description,
		"stock", product.Stock,
	).Err()
	if err != nil {
		log.Printf("警告: 插入商品到Redis失败: %v", err)
	}
	return productID, nil
}

//商店更新库存
func UpdateProductStock(rp *RedisPool, db *sql.DB, productID int, newStock int) error {
	// 更新数据库中的库存
	query := `UPDATE products SET stock = $1 WHERE productid = $2`
	result, err := db.Exec(query, newStock, productID)
	if err != nil {
		return fmt.Errorf("更新库存失败: %v", err)
	}

	// 检查是否成功更新：返回被 SQL 更新语句影响的行数
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查更新结果失败: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("商品不存在")
	}

	// 同步更新Redis缓存
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	err = rdb.HSet(context.Background(),
		fmt.Sprintf("product:%d", productID), //生成 Redis key
		"stock", newStock,
	).Err()
	if err != nil {
		log.Printf("警告: 更新Redis缓存失败: %v", err)
	}

	return nil
}
