package database

import (
	"context"
    "database/sql"
    "fmt"
    "take-out/models"
	"time"
	"log"
	
	"github.com/go-redis/redis/v8"
)

//添加商品到商店
func addProductForShop(rp *database.RedisPool, db *database.sql.DB, ShopID int, product *models.Product) (int64, error){
	//检查商店是否存在
	shopexist := db.CheckShopExist(ShopID)
	if shopexist == false {
	    return nil fmt.Errorf("商店不存在") 
	}

	//添加商品到商店
	query = `INSERT INTO products (shopid, productname, productprice, productdescription, stock)
			 VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.QueryRow(query, product.ShopID, product.ProductName, product.ProductPrice, product.ProductDescription, product.Stock)
	if err != nil {
	    return nil, fmt.Errorf("添加商品失败") 
	}

	//获取新商品ID
	ProductID, err := result.LastInsertId()
	if err != nil {
	    return nil, fmt.Errorf("获取新商品ID失败")
	}

	//更新商品
	query = `UODATE products SET stock WHERE shop = ?`
	redult, err := db.QueryRow(query, product.ShopID)
	if err != nil {
	    return nil, fmt.Errorf("更新商品失败")
	}

	//插入商品到Redis
	rdb := rp.Get()          //获取连接
	defer rp.PutClient(rdb)  //确保归还连接

	err = rp.HSet(context.Background(),
		fmt.Sprintf("product:%d", ProductID),
		map[string]interface{
			"productid": product.ProductID,
			"shopid": product.ShopID,
			"productname": product.ProductName,
			"productprice": product.ProductPrice,
			"productdescription": product.ProductDescription,
			"stock": product.Stock,
		}
	).Err()
	if err != nil {
	    return nil, fmt.Errorf("插入商品到Redis失败")
	}
	return ProductID, nil
}

//商店更新库存
func UpdateProductStock(rp *database.RedisPool, db *database.sql.DB, productID int, newStock int) error {
    // 更新数据库中的库存
    query := `UPDATE products SET stock = ? WHERE productid = ?`
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
    err = rp.HSet(context.Background(),
        fmt.Sprintf("product:%d", productID),  //生成 Redis key
        "stock", newStock,
    )
    if err != nil {
        return fmt.Errorf("更新缓存失败: %v", err)
    }

    return nil
}

