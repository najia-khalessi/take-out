package database
import (
	"take-out/models"
	"fmt"
	"database/sql"
)
//下单：通过连接Redis连接池的客户端，创建一个消息队列在redis中，将订单信息以json格式存储到消息队列中
func UserPlaceOrder(orderID, userID, shopID, riderID int, products []Product, rp *RedisPool) error {
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)

	// 构建订单信息
	order := map[string]interface{}{
		"order_id": orderID,
		"user_id":  userID,
		"shop_id":  shopID,
		"rider_id": riderID,
		"products": products, // 将商品信息添加到订单
	}

	// 将订单信息转化为 JSON 格式
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("订单序列化失败: %v", err)
	}

	// 将订单发布到 Redis 的 "order_channel" 频道
	_, err = rdb.Publish(context.Background(), "order_channel", orderJSON).Result()
	if err != nil {
		return fmt.Errorf("订单发布失败: %v", err)
	}

	fmt.Println("订单已成功发布到订单频道")
	return nil
}


// 新增函数：通知商家有新订单（需添加到 handlers 包）
func notifyShop(orderID int) error {
    // 1. 获取订单详情（参考 handleOrderStatus 逻辑）
    order, err := QueryOrderStatus(db, orderID)
    if err != nil {
        log.Printf("获取订单详情失败: orderID=%d, error=%v", orderID, err)
        return fmt.Errorf("订单查询失败")
    }

    // 2. 构建商家通知频道（参考 handNotifyNearbyRider 的频道设计）
    shopChannel := fmt.Sprintf("shop_%d", order.ShopID)
    
    // 3. 准备通知数据（参考 UserPlaceOrder 的订单结构）
    notification := map[string]interface{}{
        "type":      "new_order",
        "order_id":  orderID,
        "user_id":   order.UserID,
        "timestamp": time.Now().Unix(),
    }
    notifJSON, _ := json.Marshal(notification)

    // 4. 发布通知到商家频道（参考 Redis 发布逻辑）
    rdb := rp.GetClient()
    defer rp.PutClient(rdb)
    
    if _, err := rdb.Publish(context.Background(), shopChannel, notifJSON).Result(); err != nil {
        log.Printf("商家通知发布失败: shopID=%d, error=%v", order.ShopID, err)
        return fmt.Errorf("通知发送失败")
    }

    // 5. 更新订单系统状态（可选）
    _ = UpdateOrderStatus(db, orderID, "等待商家确认") // 忽略错误继续流程
    
    log.Printf("已通知商家: shopID=%d, orderID=%d", order.ShopID, orderID)
    return nil
}

//添加订单频道消费者，在服务启动时初始化订阅
func startOrderConsumer(rp *RedisPool) {
    rdb := rp.GetClient()
    pubsub := rdb.Subscribe(context.Background(), "order_channel")
    defer pubsub.Close()
    
    for msg := range pubsub.Channel() {
        var order map[string]interface{}
        if err := json.Unmarshal([]byte(msg.Payload), &order); err != nil {
            log.Printf("订单解析失败: %v", err)
            continue
        }
        
        // 实际业务处理（如自动分配商家）
        orderID := int(order["order_id"].(float64))
        log.Printf("处理新订单: ID=%d", orderID)
        
        // 示例：自动分配最近商家
        if shopID, err := AssignNearestShop(order); err == nil {
            UpdateOrderShop(orderID, shopID) // 更新数据库
        }
    }
}


// 插入订单到数据库，使用事务
func InsertOrder(db *sql.DB, order *Order) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("无法开始事务: %v", err)
	}
	defer tx.Rollback()

	query := "INSERT INTO orders (user_id, shop_id, status, total_price) VALUES (?, ?, ?, ?)"
	result, err := tx.Exec(query, order.UserID, order.ShopID, order.OrderStatus, order.TotalPrice)
	if err != nil {
		return 0, fmt.Errorf("订单插入失败: %v", err)
	}

	orderID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取订单ID失败: %v", err)
	}
	if err := tx.Commit(); err != nil {
        return 0, fmt.Errorf("事务提交失败: %v", err)
    }

	return orderID, nil
}

// 更新订单状态，使用事务
func UpdateOrderStatus(db *sql.DB, OrderID int, OrderStatus string) error {
	// 开始一个事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("无法开始事务: %v", err)
	}
	defer tx.Rollback()
	// 执行更新订单状态的 SQL 查询
	query := "UPDATE orders SET status = ? WHERE order_id = ?"
	_, err = tx.Exec(query, OrderStatus, OrderID)
	if err != nil {
		return fmt.Errorf("订单状态更新失败: %v", err)
	}
	if err := tx.Commit(); err != nil {
        return 0, fmt.Errorf("事务提交失败: %v", err)
    }

	return nil
}

// 查询订单状态
func QueryOrderStatus(db *sql.DB, orderID int) (*Order, error) {
	var order Order
	query := `SELECT order_id, rider_id, shop_id, product_id, order_time, total_price, order_status FROM orders WHERE order_id = ?`
	row := db.QueryRow(query, orderID)
	err := row.Scan(&order.OrderID, &order.RiderID, &order.ShopID, &order.ProductID, &order.OrderTime, &order.TotalPrice, &order.OrderStatus)
	// 一般返回username, shopname而不是ID, 这里为了方便测试而用ID
	if err != nil {
		return nil, err
	}
	return &order, nil
}

//骑手接单
func AcceptOrderTx(db *sql.DB, OrderID int, RiderID int) error {
		tx, err = db.Begin()
		if err != nil {
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback() 
		//检查订单状态
		var currentStatus string
		var currentRiderID sql.NullInt64    //不直接用int，是因为NULL 值会导致扫描错误，无法区分 0 和 NULL
		statusQuery := `SELECT orderstatus FROM orders WHERE orderid=? FOR UPDATE`  //FOR UPDATA作用：行级锁定，事务隔离，避免并发问题
		err := tx.QueryRow(statusQuery, OrderID, RiderID).Scan(&currentStatus)       //          优点：数据一致性，并发控制，业务安全
		if err != nil {
			return fmt.Errorf("获取订单出错：%v",err)
		}
		if currentStatus == "confirmed" {   //已出餐
		
			return fmt.Errorf("订单已出餐，不允许接单")
		}
		if currentRiderID.Valid {           //valid表示是否为NULL
			return fmt.Errorf("订单已被其他骑手接单")
		}
		//更新订单状态
		_, err = tx.Exec(`UPDATE orders SET riderid = ?,orderstatus = "delivering" WHERE orderid = ? `, RiderID, OrderID)
		if err != nil {
			return fmt.Errorf("更新订单状态失败: %v", err)
		}

		//更新聊天群组，添加骑手
		_, err = tx.QueryRow(`UPDATE groups SET riderid = ? WHERE orderid = ?`, RiderID, OrderID)
		if err != nil {
			return fmt.Errorf("更新聊天群组失败: %v", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交事务失败: %v", err)
		}
		return nil
	}


//完成订单
func CompleteOrderTx(db *sql.DB, OrderID int, RiderID int) error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback() 
		var currentStatus string
		statusQuery := `SELECT orderstatus FROM orders WHERE orderid=? FOR UPDATE`
		_, err = tx.QueryRow(statusQuery, OrderID).Scan(&currentStatus)
		if err != nil {
			return fmt.Errorf("查询订单状态失败：%v", err)
		}
		if currentStatus != "completed" {
			return fmt.Errorf("订单还未送达...")
		}

		//更新订单状态
		_, err = tx.Exec(`UPDATE orders SET orderstatus = "completed" WHERE orderid = ?`, OrderID)
		if err != nil {
			return fmt.Errorf("更新订单状态失败：%v", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交事务失败: %v", err)
		}
		return nil
	}

// 取消订单
func DeleteOrder(db *sql.DB, OrderID int, ProductID int) error {
		tx, err := db.Begin()
		if err != nil {
		    return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()
		//检查订单状态
		var currentStatus string
		statusQuery := `SELECT orderstatus FROM orders WHERE orderid = ? FOR UPDATE`
		err := tx.QueryRow(statusQuery, OrderID).Scan(&currentStatus)
		if err != nil {
			return fmt.Errorf("获取订单状态失败：%v", err)
		}

		if currentStatus == "completed" || currentStatus == "canceled" {
			return fmt.Errorf("订单已完成，无法取消")
		}


		//恢复商品库存
		_, err = tx.Exec(`UPDATE products SET stock = stock + 1 WHERE productid = (SELECT productid FROM orders WHERE orderid = ?)`, OrderID)
		if err != nil {
			return fmt.Errorf("恢复商品库存失败：%v", err)
		}

		//直接更新订购单状态
		_, err = tx.QueryRow(`UPDATE orders SET orderstatus = "canceled" WHERE orderid = ?`, OrderID)
		if err != nil {
			return fmt.Errorf("更新订单状态失败：%v", err)
		}
		return nil
	}


