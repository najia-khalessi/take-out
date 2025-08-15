//程序入口，初始化路由、数据库连接、启动服务器
package main

import (
	"database/sql"
	"net/http"
	"take-out/database"
	"take-out/handlers"
	"take-out/logging"
	"take-out/monitoring"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	db *sql.DB
	rp *database.RedisPool
)

func main() {
	var err error
	// 初始化日志
	logging.Init()

	// 初始化 Redis 连接池
	rp, err = database.InitRedis()
	if err != nil {
		logging.Error("Redis初始化失败", logrus.Fields{"error": err})
	}

	// 初始化数据库连接
	db, err = database.InitDB()
	if err != nil {
		// 连接失败时启动重试机制
		db, err = database.RetryConnect(3)
		if err != nil {
			logging.Error("数据库连接失败", logrus.Fields{"error": err})
		}
	}
	defer db.Close()

	//启动数据库监控
	go database.StartDBMonitor(db, 5*time.Minute) // 每5分钟监控一次

	// 启动后台任务
	go handlers.StartOrderConsumer(rp)
	go database.StartWeeklyCleanUpScheduler(db)

	// 暴露 /metrics 接口
	http.Handle("/metrics", handlers.LoggingMiddleware(monitoring.MetricsHandler()))

	// 无需Token验证的路由 - 使用/auth路径避免冲突
	http.Handle("/api/auth/user/register", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleUserRegister(db, rp))))
	http.Handle("/api/auth/user/login", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleUserLogin(db))))
	http.Handle("/api/auth/shop/register", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleShopRegister(db, rp))))
	http.Handle("/api/auth/shop/login", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleShopLogin(db))))
	http.Handle("/api/auth/rider/register", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleRiderRegister(db, rp))))
	http.Handle("/api/auth/rider/login", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleRiderLogin(db))))
	http.Handle("/api/auth/refresh", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleRefreshToken(rp))))

	// 用户路由组 - 需要认证，使用精确路径避免与auth路由冲突
	userRoutes := http.NewServeMux()
	userRoutes.Handle("/shops", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleGetShops(db))))
	userRoutes.Handle("/products", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleShopProducts(db, rp))))
	userRoutes.Handle("/order", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleOrder(db, rp))))
	userRoutes.Handle("/order/status", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleOrderStatus(db, rp))))
	userRoutes.Handle("/nearby-shops", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleNearbyShops(db, rp))))
	// IM 路由
	userRoutes.Handle("/im/send", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleSendMessage(db, rp))))
	userRoutes.Handle("/im/messages", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleGetMessages(db, rp))))
	// 评价路由
	userRoutes.Handle("/review/create", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.CreateReview(db, rp))))
	userRoutes.Handle("/review/update", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.UpdateReview(db, rp))))
	http.Handle("/api/user/", handlers.LoggingMiddleware(handlers.AuthenticateToken(rp)(http.StripPrefix("/api/user", userRoutes))))

	// 商家路由组 - 需要认证
	shopRoutes := http.NewServeMux()
	shopRoutes.Handle("/add_product", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleAddProduct(db, rp))))
	shopRoutes.Handle("/update_stock", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleUpdateProductStock(db, rp))))
	shopRoutes.Handle("/accept_order", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleAcceptOrder(db, rp))))
	shopRoutes.Handle("/publish_order", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandlePublishDeliveryOrder(db, rp))))
	// 评价路由
	shopRoutes.Handle("/reviews", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.GetShopReviews(db, rp))))
	shopRoutes.Handle("/review/reply", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.ReplyToReview(db, rp))))
	shopRoutes.Handle("/review/analytics", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.GetReviewAnalytics(db, rp))))
	http.Handle("/api/shop/", handlers.LoggingMiddleware(handlers.AuthenticateTokenShop(rp)(http.StripPrefix("/api/shop", shopRoutes))))

	// 骑手路由组 - 需要认证
	riderRoutes := http.NewServeMux()
	riderRoutes.Handle("/grab", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleRiderGrabOrder(db, rp))))
	riderRoutes.Handle("/complete", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.HandleCompleteOrder(db))))
	// 评价路由
	riderRoutes.Handle("/confirm_delivery", handlers.LoggingMiddleware(monitoring.PrometheusMiddleware(handlers.RiderConfirmDelivery(db, rp))))
	http.Handle("/api/rider/", handlers.LoggingMiddleware(handlers.AuthenticateTokenRider(rp)(http.StripPrefix("/api/rider", riderRoutes))))

	// 启动服务器
	logging.Info("服务器启动，端口 :8080", nil)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logging.Error("服务器启动失败", logrus.Fields{"error": err})
	}
}