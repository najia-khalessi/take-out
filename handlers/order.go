package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"take-out/database"
	"take-out/models"
	"time"

	"github.com/redis/go-redis/v9"
)

//在服务启动时初始化消费者
func StartOrderConsumer(rp *database.RedisPool) {
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)

	pubsub := rdb.Subscribe(context.Background(), "order_channel")
	ch := pubsub.Channel()

	for msg := range ch {
		var order models.Order
		if err := json.Unmarshal([]byte(msg.Payload), &order); err != nil {
			log.Printf("订单解析失败: %v", err)
			continue
		}
		// 实际业务处理（如分配商家）
		log.Printf("收到新订单: ID=%d, 用户=%d", order.OrderID, order.UserID)
	}
}

//订单提交并且更新Redis缓存
func HandleOrder(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}
		var order models.Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		// 查询商品价格
		var productPrice float64
		if err := db.QueryRow("SELECT price FROM products WHERE productid = ?", order.ProductID).Scan(&productPrice); err != nil {
			http.Error(w, "商品不存在", http.StatusBadRequest)
			return
		}
		// 查询骑手配送费
		var deliveryFee float64
		if err := db.QueryRow("SELECT delivery_fee FROM riders WHERE riderid = ?", order.RiderID).Scan(&deliveryFee); err != nil {
			deliveryFee = 0 // 没有骑手时默认0元
		}
		order.DeliveryFee = deliveryFee
		order.TotalPrice = productPrice + deliveryFee

		// 设置初始订单状态
		order.OrderStatus = "商家待确认"

		// 插入订单到数据库
		orderID, err := database.InsertOrder(db, &order)
		if err != nil {
			http.Error(w, fmt.Sprintf("订单插入失败: %v", err), http.StatusInternalServerError)
			return
		}

		order.OrderID = int(orderID)

		// 更新 Redis 缓存
		jsonData, _ := json.Marshal(order)
		database.SetToCache(rp, fmt.Sprintf("order_status_%d", order.OrderID), string(jsonData), time.Hour)

		// 订单已成功插入数据库并获得了有效的 orderID，Redis 缓存已更新，确保数据一致性，发布到消息队列
		err = database.UserPlaceOrder(order.OrderID, order.UserID, order.ShopID, 0, []models.Product{}, rp)
		if err != nil {
			log.Printf("发布订单到消息队列失败: %v", err)
		}

		// 返回订单详情和订单ID
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(order)
	}
}

// 查询订单状态
func HandleOrderStatus(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderIDStr := r.URL.Query().Get("order_id")
		orderID, err := strconv.Atoi(orderIDStr)
		if err != nil {
			http.Error(w, "无效的订单ID", http.StatusBadRequest)
			return
		}

		cacheKey := fmt.Sprintf("order_status_%d", orderID)
		data, err := database.GetFromCache(rp, cacheKey)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(data))
			return
		}

		// 缓存未命中，从数据库查询订单状态
		order, err := database.QueryOrderStatus(db, orderID)
		if err != nil {
			http.Error(w, fmt.Sprintf("订单不存在: %v", err), http.StatusInternalServerError)
			return
		}

		jsonData, err := json.Marshal(order)
		if err != nil {
			http.Error(w, "JSON 序列化错误", http.StatusInternalServerError)
			return
		}
		database.SetToCache(rp, cacheKey, string(jsonData), time.Hour)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

// 商家接单并确认订单
func HandleAcceptOrder(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}
		// 定义请求体结构
		var acceptRequest struct {
			OrderID int `json:"order_id"`
		}

		// 解析请求体
		if err := json.NewDecoder(r.Body).Decode(&acceptRequest); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		// 更新订单状态为 "商家已接单"
		err := database.UpdateOrderStatus(db, acceptRequest.OrderID, "商家已接单")
		if err != nil {
			http.Error(w, fmt.Sprintf("更新订单状态失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 查询订单状态
		order, err := database.QueryOrderStatus(db, acceptRequest.OrderID)
		if err != nil {
			http.Error(w, fmt.Sprintf("查询订单失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 创建聊天群组
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "开启事务失败", http.StatusInternalServerError)
			return
		}
		err = database.CreateGroup(map[string]interface{}{
			"order_id": order.OrderID,
			"user_id":  order.UserID,
			"shop_id":  order.ShopID,
			"rider_id": order.RiderID,
		}, rp, db, tx) // 或者传递一个有效的事务对象
		if err != nil {
			tx.Rollback()
			http.Error(w, fmt.Sprintf("创建聊天群组失败: %v", err), http.StatusInternalServerError)
			return
		}
		tx.Commit()

		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "订单已接单"})
	}
}

// 商家发布外卖订单
func HandlePublishDeliveryOrder(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查请求方法是否为 POST
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		// 定义请求体结构
		var publishRequest struct {
			OrderID int `json:"order_id"`
		}

		// 解析请求体
		if err := json.NewDecoder(r.Body).Decode(&publishRequest); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		// 更新订单状态为 "已发布跑腿订单"
		err := database.UpdateOrderStatus(db, publishRequest.OrderID, "已发布跑腿订单")
		if err != nil {
			http.Error(w, fmt.Sprintf("更新订单状态失败: %v", err), http.StatusInternalServerError)
			return
		}
		// 将订单发布到公共大厅队列供骑手抢单
		rdb := rp.GetClient()
		defer rp.PutClient(rdb)
		orderInfo := map[string]interface{}{
			"order_id":     publishRequest.OrderID,
			"order_status": "已发布跑腿订单",
			"order_time":   time.Now().Unix(),
		}
		orderJSON, _ := json.Marshal(orderInfo)
		rdb.Publish(context.Background(), "public_hall", orderJSON)

		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "跑腿订单已发布"})
	}
}

// 随机选择一个骑手并通知新订单
func HandNotifyNearbyRider(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 解析请求参数（假设请求体中有订单的经纬度信息）
		var order map[string]interface{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&order); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		// 获取订单的经纬度信息
		orderLat := order["latitude"].(float64)
		orderLon := order["longitude"].(float64)
		maxDistance := 5000 // 最大距离 5 公里，单位米

		// 获取 Redis 客户端
		rdb := rp.GetClient()
		defer rp.PutClient(rdb)

		// 查询数据库获取附近的骑手ID（登录状态）
		query := `
        SELECT rider_id
        FROM riders
        WHERE rider_status = "online"
        AND ST_Distance_Sphere(point(rider_longitude, rider_latitude), point(?, ?)) < ?
    `
	
rows, err := db.Query(query, orderLon, orderLat, maxDistance)
		if err != nil {
			http.Error(w, fmt.Sprintf("获取附近骑手列表失败: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// 收集附近的骑手ID
		var riderIDs []int
		for rows.Next() {
			var riderID int
			if err := rows.Scan(&riderID); err == nil {
				riderIDs = append(riderIDs, riderID)
			}
		}
		if len(riderIDs) == 0 {
			http.Error(w, "没有找到附近的骑手", http.StatusNotFound)
			return
		}

		// 随机选择一名骑手
		rand.Seed(time.Now().UnixNano())

		for len(riderIDs) > 0 {
			riderIndex := rand.Intn(len(riderIDs))
			selectedRiderID := riderIDs[riderIndex]

			// 通知骑手新订单
			notifyChannel := fmt.Sprintf("rider_%d", selectedRiderID)
			orderJSON, _ := json.Marshal(order)
			_, err := rdb.Publish(context.Background(), notifyChannel, orderJSON).Result()
			if err != nil {
				fmt.Printf("通知骑手失败: %v\n", err)
				continue
			}

			// 等待骑手响应
			response, err := waitForRiderResponse(selectedRiderID)
			if err != nil || response != "accept" {
				// 若骑手拒绝接单，继续通知下一个骑手
				fmt.Printf("骑手 %d 拒绝订单，转派其他骑手\n", selectedRiderID)
				// 移除已通知的骑手
				riderIDs = append(riderIDs[:riderIndex], riderIDs[riderIndex+1:]...)
			} else {
				fmt.Printf("骑手 %d 接受订单\n", selectedRiderID)
				// 返回成功响应
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "骑手 %d 接受订单", selectedRiderID)
				return
			}
		}

		// 如果没有骑手接受订单
		http.Error(w, "所有骑手拒绝订单", http.StatusServiceUnavailable)
	}
}

// 模拟等待骑手响应接单请求
func waitForRiderResponse(riderID int) (string, error) {
	// 假设客户端通过某个途径返回接受或拒绝的状态
	// 为简单起见，这里模拟等待响应并返回随机状态
	time.Sleep(2 * time.Second) // 等待时间
	responseOptions := []string{"accept", "reject"}
	return responseOptions[rand.Intn(len(responseOptions))], nil
}

// 获取订单列表供骑手选择
func GetOrderListFromMQ(rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取 Redis 客户端
		rdb := rp.GetClient()
		defer rp.PutClient(rdb)

		// 消费 MQ 获取订单列表
		var orderList []map[string]interface{}

		for i := 0; i < 10; i++ { // 假设每次获取最多 10 个订单
			orderJSON, err := rdb.LPop(context.Background(), "public_hall").Result()
			if err == redis.Nil {
				break // 没有更多订单
			} else if err != nil {
				http.Error(w, fmt.Sprintf("订单获取失败: %v", err), http.StatusInternalServerError)
				return
			}

			var order map[string]interface{}
			if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
				http.Error(w, fmt.Sprintf("订单解析失败: %v", err), http.StatusInternalServerError)
				return
			}
			orderList = append(orderList, order)
		}

		// 返回订单列表
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(orderList)
	}
}

// 处理骑手抢单请求，保证事务处理
func HandleRiderGrabOrder(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only supports POST method", http.StatusMethodNotAllowed)
			return
		}

		var requestData struct {
			OrderID int `json:"order_id"`
			RiderID int `json:"rider_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err := database.AcceptOrderTx(db, requestData.OrderID, requestData.RiderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Order accepted successfully"})
	}
}

// 处理骑手完成订单的请求
func HandleCompleteOrder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var completeRequest struct {
			OrderID int `json:"order_id"`
			RiderID int `json:"rider_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&completeRequest); err != nil {
			http.Error(w, "请求体解析错误", http.StatusBadRequest)
			return
		}

		// 更新订单状态为 "已完成"
		err := database.CompleteOrderTx(db, completeRequest.OrderID, completeRequest.RiderID)
		if err != nil {
			http.Error(w, fmt.Sprintf("更新订单状态失败: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "订单已完成"})
	}
}