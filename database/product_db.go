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
// 秒杀商品结构
type SeckillProduct struct {
	ProductID   int
	TotalStock  int
	SeckillStock int
	StartTime   time.Time
	EndTime     time.Time
}
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

// 初始化秒杀商品（活动开始前调用）
func InitSeckillProduct(rp *RedisPool, db *sql.DB, sp SeckillProduct) error {
	// 1. 保存到数据库
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	_, err = tx.Exec(`
		INSERT INTO seckill_products 
		(product_id, total_stock, seckill_stock, start_time, end_time)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		seckill_stock = VALUES(seckill_stock),
		start_time = VALUES(start_time),
		end_time = VALUES(end_time)`,
		sp.ProductID, sp.TotalStock, sp.SeckillStock, sp.StartTime, sp.EndTime)
	
	if err != nil {
		return err
	}
	
	// 2. 预热Redis库存
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	
	key := fmt.Sprintf("seckill:%d", sp.ProductID)
	err = rdb.Set(context.Background(), key, sp.SeckillStock, 0).Err()
	if err != nil {
		return err
	}
	
	return tx.Commit()
}

// Redis预减库存（原子操作），减少秒杀库存
func PreReduceSeckillStock(rp *RedisPool, productID int) (bool, error) {
	// 获取Redis客户端
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	
	// 创建上下文
	ctx := context.Background()
	// 构造Redis键
	key := fmt.Sprintf("seckill:%d", productID)
	
	// 使用Lua脚本保证原子性
	script := redis.NewScript(`
		local stock = tonumber(redis.call('get', KEYS[1]))
		if stock and stock > 0 then
			redis.call('decr', KEYS[1])
			return 1
		end
		return 0
	`)
	
	// 执行Lua脚本
	count, err := script.Run(ctx, rdb, []string{key}).Int()
	if err != nil {
		return false, err
	}
	
	// 返回结果
	return count == 1, nil
}

// 回滚Redis库存（当后续步骤失败时调用）
func RollbackSeckillStock(rp *RedisPool, productID int) error {
	// 从Redis连接池中获取一个Redis客户端
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)

	ctx := context.Background()
	// 格式化秒杀库存的Redis键
	key := fmt.Sprintf("seckill:%d", productID)
	
	// 将秒杀库存的值加1
	//Redis 的INCR命令用于将存储在键中的数字值递增 1
	_, err := rdb.Incr(ctx, key).Result()
	// 返回错误
	return err
}