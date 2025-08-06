//PostgreSQL数据库连接、基础CRUD操作
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"take-out/logging"
	"time"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const (
	maxRetries = 3
	timeout    = 5 * time.Second
)

//全局数据库连接
var DB *sql.DB

func InitDB() (*sql.DB, error) {
	// 从 .env 文件加载环境变量
	err := godotenv.Load("config.env")
	if err != nil {
		logging.Warn("加载 .env 文件失败", logrus.Fields{"error": err})
	}

	// 从环境变量获取数据库配置
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// 添加context支持超时控制
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() //防止泄露

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("数据库驱动初始化失败:%v", err)
	}
	// 设置数据库连接池的参数
	db.SetMaxOpenConns(100) //设置最大打开连接数
	db.SetMaxIdleConns(10)   // 保持10个空闲连接
	db.SetConnMaxLifetime(time.Hour) // 连接最多重用1小时

	//测试连接，如果 5 秒内未完成连接，会返回超时错误
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	logging.Info("数据库连接成功并已配置连接池", nil)
	return db, nil
}

// RetryConnect 函数用于重试连接数据库，最多重试 maxRetries 次
func RetryConnect(maxRetries int) (*sql.DB, error) {
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
		logging.Warn("重试连接失败", logrus.Fields{"retry": i + 1, "error": err})
		time.Sleep(time.Second * time.Duration(i+1))
	}

	// 如果达到最大重试次数，则返回 nil 和错误信息
	return nil, fmt.Errorf("达到最大重试次数(%d): %w", maxRetries, err)
}
