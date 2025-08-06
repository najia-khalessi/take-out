//Redis连接、缓存操作、消息队列

package database

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

//定义连接池类型
type RedisPool struct {
	pool *sync.Pool
}

func InitRedis() (*RedisPool, error) {
	// 从环境变量获取 Redis 配置
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD") // 默认为空
	redisDBStr := os.Getenv("REDIS_DB")
	if redisDBStr == "" {
		redisDBStr = "0"
	}
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		return nil, err
	}

	pool := &sync.Pool{
		New: func() interface{} {
			return redis.NewClient(&redis.Options{
				Addr:         redisAddr,
				Password:     redisPassword,
				DB:           redisDB,
				PoolSize:     100, //连接池大小
				MinIdleConns: 5,   //最小空闲连接数
				DialTimeout:  10 * time.Second,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			})
		},
	}
	return &RedisPool{pool: pool}, nil
}

//从获 Redis 池取一个客户端
func (rp *RedisPool) GetClient() *redis.Client {
	return rp.pool.Get().(*redis.Client)
}

// 将 Redis 客户端放回连接池
func (rp *RedisPool) PutClient(rdb *redis.Client) {
	rp.pool.Put(rdb)
}

// 从 Redis 获取数据
func GetFromCache(rp *RedisPool, key string) (string, error) {
	rdb := rp.GetClient()   // 从连接池获取一个 Redis 客户端
	defer rp.PutClient(rdb) // 使用完后归还到连接池
	return rdb.Get(ctx, key).Result()
}

// 将数据写入 Redis
func SetToCache(rp *RedisPool, key string, value string, expiration time.Duration) error {
	rdb := rp.GetClient()   // 从连接池获取 Redis 客户端
	defer rp.PutClient(rdb) // 使用完后归还到连接池
	return rdb.Set(ctx, key, value, expiration).Err()
}

// 删除 Redis 中的键
func DeleteFromCache(rp *RedisPool, key string) error {
	rdb := rp.GetClient()   // 从连接池获取 Redis 客户端
	defer rp.PutClient(rdb) // 使用完后归还到连接池
	return rdb.Del(ctx, key).Err()
}