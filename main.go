//程序入口，初始化路由、数据库连接、启动服务器
package main

import (
    "log"
    "take-out/database"  // 修改为相对路径
)
var (
	rp *RedisPool // Redis连接池
	db *sql.DB    // MySQL数据库连接
)

// 服务启动时调用
go startOrderConsumer(redisPool)

func main() {
    if err := database.InitRedis(); err != nil {
        log.Fatalf("Redis初始化失败: %v", err)
    }
    // 初始化数据库连接
    db, err := database.InitDB()
    if err != nil {
        // 连接失败时启动重试机制
        db, err = database.RetryConnect(database.maxRetries)
        if err != nil {
            log.Fatal("数据库连接失败:", err)
        }
    }
    defer db.Close()
    
    //启动数据库监控
    database.StartDBMonitor(db, 5*time.Minute) // 每5分钟监控一次
    
	// 初始化一致性哈希
    hashConsistent = common.NewConsistent()
    for _, node := range config.SeckillNodes {
        hashConsistent.Add(node)
    }
    
    // 获取本机IP
    localIp, err := common.GetEntranceIp()
    if err != nil {
        log.Fatal(err)
    }
    localHost = localIp
  
	// 启动服务器
    if err := StartServer(); err != nil {
        log.Fatal("服务器启动失败:", err)
    }

    //Gin框架
    router := gin.Default()
    router.GET("/shops", handleGetShops)  // 处理获取店铺列表的请求
    router.GET("/shop/products", handleShopProducts )  // 处理获取店铺商品列表的请求
    router.GET("/shop/orders", handleNearbyShops)  // 处理获取附近店铺的请求

    http.HandleFunc("/protected", authenticateToken(protectedEndpoint))                // 验证用户
	http.HandleFunc("/protected/shop", authenticateTokenShop(rp, protectedEndpoint))   // 验证商家
	http.HandleFunc("/protected/rider", authenticateTokenRider(rp, protectedEndpoint)) // 验证骑手

	http.HandleFunc("/user/register", handleRegister) // 用户注册
	http.HandleFunc("/user/login", handleLogin)       // 用户登录

	http.HandleFunc("/shops", handleGetShops)           // 获取商家列表
	http.HandleFunc("/shops", handleNearbyShops)        // 获取附近商家
	http.HandleFunc("/products", handleShopProducts)    // 查询商家商品
	http.HandleFunc("/order", handleOrder)              // 用户提交订单
	http.HandleFunc("/order/status", handleOrderStatus) // 查询订单状态

	http.HandleFunc("/shop/register", handleRegisterShop)              // 商家注册
	http.HandleFunc("/shop/login", handleLoginShop)                    // 商家登录
	http.HandleFunc("/shop/add_product", handleAddProductForShop)      // 商家添加商品
	http.HandleFunc("/shop/accept_order", handleAcceptOrder)           // 商家接单+确认订单
	http.HandleFunc("/shop/publish_order", handlePublishDeliveryOrder) // 商家发布订单

	
	http.HandleFunc("/rider/apply", handleApplyForRider)         // 骑手身份申请
	http.HandleFunc("/notify", handNotifyNearbyRider) // 系统随机通知骑手
	http.HandleFunc("/rider/grab", handleRiderGrabOrder) // 骑手抢单
	http.HandleFunc("/rider/complete", handleCompleteOrder)      // 骑手完成订单

	http.HandleFunc("/im/send", handleSendMessage(db, rp))     // 发送群组消息
	http.HandleFunc("/im/messages", handleGetMessages(db, rp)) // 获取群组消息

	// 启动每周清理调度器
	go StartWeeklyCleanUpScheduler(db)

	log.Println("服务器启动，端口 :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}

    // 启动秒杀订单消费者
    go startSeckillOrderConsumer()
    
    // 注册秒杀路由
    http.HandleFunc("/seckill/order", 
        SeckillAuthMiddleware(handleSeckillOrder))

	http.HandleFunc("/refresh", handleRefreshToken) // 新增路由，用于刷新token
}