//MySQL数据库连接、基础CRUD操作
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	_ "github.com/go-sql-driver/mysql"
)

const (
    maxRetries = 3
    timeout    = 5 * time.Second
)

//全局数据库连接
var DB *sql.DB
func InitDB() (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        "root", "123456", "localhost", 3306, "utf8") 

	// 添加context支持超时控制
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()   //防止泄露

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("数据库连接失败:%v\n", err)
	}
	// 设置数据库连接池的参数
	db.SetMaxOpenConns(100)  //设置最大打开连接数
	db.SetMaxIdleConns(10)   // 保持10个空闲连接
	db.SetConnMaxLifetime(time.Hour)  // 连接最多重用1小时

	//测试连接，如果 5 秒内未完成连接，会返回超时错误
	if err = db.PingContext(ctx); err != nil {
		log.Fatalf("数据库连接测试失败:%v\n, 尝试重连并监控", err)
		//异常处理：数据库重连机制+监控
		for i := 0; i < maxRetries; i++ {
        	db, err = InitDB()
       		if err == nil {
				if db != nil{
                	stats := db.Stats()
					log.Printf("=== 数据库重连成功，连接池状态 ===")
    				log.Printf("打开连接数: %d", stats.OpenConnections)
					log.Printf("使用中连接数: %d", stats.InUse)
					log.Printf("空闲连接数: %d", stats.Idle)
				}
				return db, nil
			}
			time.Sleep(time.Second * time.Duration(i+1))
		}
	return nil, fmt.Errorf("达到最大重试次数(%d): %w", maxRetries, err)
}
fmt.Println("数据库连接成功并以配置连接池")
return db, nil
}

// 重试机制
// RetryConnect 函数用于重试连接数据库，最多重试 maxRetries 次
func RetryConnect(maxRetries int) (*sql.DB, error) {
    // 定义数据库连接变量
    var db *sql.DB
    var err error

    for i := 0; i < maxRetries; i++ {
        // 初始化数据库连接
        db, err = InitDB()
        // 如果连接成功，则返回数据库连接和 nil 错误
        if err == nil {
            logDBStats(db)
            return db, nil
        }
        // 如果连接失败，则打印错误信息，并等待一段时间后重试
        log.Printf("第 %d 次重试连接失败: %v", i+1, err)
        time.Sleep(time.Second * time.Duration(i+1))
    }
    
    // 如果达到最大重试次数，则返回 nil 和错误信息
    return nil, fmt.Errorf("达到最大重试次数(%d): %w", maxRetries, err)
}

// 监控mysql连接池
func logDBStats(db *sql.DB) {
    if db != nil {
        stats := db.Stats()
        log.Printf("=== 数据库连接池状态 ===")
        log.Printf("打开连接数: %d", stats.OpenConnections)
        log.Printf("使用中连接数: %d", stats.InUse)
        log.Printf("空闲连接数: %d", stats.Idle)
    }
}
