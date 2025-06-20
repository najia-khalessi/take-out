//Redis连接、缓存操作、消息队列

package database

import (
	"fmt"
	"context"
	"time"
	"github.com/go-redis/redis/v8"
	"sync"
)

var ctx = context.Background()

//定义连接池类型
type RedisPool struct {
	pool *sync.Pool
}
func InitRedisPool() *RedisPool {
	return &RedisPool {                    
		pool: &sync.Pool{              
			New: func() interface{} {                        
				return redis.NewClient(&redis.Options{         
						Addr:  "localhost:6379",
						Password: "",
						DB:     0,
						PoolSize:100,  //连接池大小
						MinIdleConns:5,  //最小空闲连接数
						DialTimeout:10 * time.Second,
						ReadTimeout:30 * time.Second,
						WriteTimeout:30 * time.Second,
						})
				}
			}
		}
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
