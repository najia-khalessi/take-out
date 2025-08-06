package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"take-out/logging"
	"take-out/models"
	"take-out/monitoring"
	"time"

	"github.com/sirupsen/logrus"
)

// todo 加上登陆状态
//下单：通过连接Redis连接池的客户端，创建一个消息队列在redis中，将订单信息以json格式存储到消息队列中
func UserPlaceOrder(orderID, userID, shopID, riderID int, products []models.Product, rp *RedisPool) error {
	logging.Info("Placing order", logrus.Fields{"orderID": orderID, "userID": userID, "shopID": shopID})
	err := monitoring.RecordRedisTime("UserPlaceOrder", func() error {
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
		return err
	})
	if err != nil {
		logging.Error("Failed to place order", logrus.Fields{"error": err, "orderID": orderID})
		return fmt.Errorf("订单发布失败: %v", err)
	}

	logging.Info("Order placed successfully", logrus.Fields{"orderID": orderID})
	return nil
}

// 新增函数：通知商家有新订单（需添加到 handlers 包）
func notifyShop(db *sql.DB, rp *RedisPool, orderID int) error {
	logging.Info("Notifying shop", logrus.Fields{"orderID": orderID})
	// 1. 获取订单详情（参考 handleOrderStatus 逻辑）
	order, err := QueryOrderStatus(db, orderID)
	if err != nil {
		logging.Error("Failed to query order status", logrus.Fields{"error": err, "orderID": orderID})
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
	err = monitoring.RecordRedisTime("notifyShop", func() error {
		rdb := rp.GetClient()
		defer rp.PutClient(rdb)
		_, err := rdb.Publish(context.Background(), shopChannel, notifJSON).Result()
		return err
	})
	if err != nil {
		logging.Error("Failed to notify shop", logrus.Fields{"error": err, "shopID": order.ShopID, "orderID": orderID})
		return fmt.Errorf("通知发送失败")
	}

	// 5. 更新订单系统状态（可选）
	_ = UpdateOrderStatus(db, orderID, "等待商家确认") // 忽略错误继续流程

	logging.Info("Shop notified successfully", logrus.Fields{"shopID": order.ShopID, "orderID": orderID})
	return nil
}

// 插入订单到数据库，使用事务
func InsertOrder(db *sql.DB, order *models.Order) (int64, error) {
	logging.Info("Inserting order", logrus.Fields{"userID": order.UserID, "shopID": order.ShopID})
	var orderID int64
	err := monitoring.RecordDBTime("InsertOrder", func() error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("无法开始事务: %v", err)
		}
		defer tx.Rollback()

		query := "INSERT INTO orders (user_id, shop_id, orderstatus, totalprice, delivery_fee) VALUES ($1, $2, $3, $4, $5) RETURNING orderid"
		err = tx.QueryRow(query, order.UserID, order.ShopID, order.OrderStatus, order.TotalPrice, order.DeliveryFee).Scan(&orderID)
		if err != nil {
			return fmt.Errorf("订单插入失败: %v", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("事务提交失败: %v", err)
		}
		return nil
	})
	if err != nil {
		logging.Error("Failed to insert order", logrus.Fields{"error": err, "userID": order.UserID, "shopID": order.ShopID})
		return 0, err
	}
	logging.Info("Order inserted successfully", logrus.Fields{"orderID": orderID})
	return orderID, nil
}

// 更新订单状态，使用事务
func UpdateOrderStatus(db *sql.DB, OrderID int, OrderStatus string) error {
	logging.Info("Updating order status", logrus.Fields{"orderID": OrderID, "status": OrderStatus})
	err := monitoring.RecordDBTime("UpdateOrderStatus", func() error {
		// 开始一个事务
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("无法开始事务: %v", err)
		}
		defer tx.Rollback()
		// 执行更新订单状态的 SQL 查询
		query := "UPDATE orders SET orderstatus = $1 WHERE order_id = $2"
		_, err = tx.Exec(query, OrderStatus, OrderID)
		if err != nil {
			return fmt.Errorf("订单状态更新失败: %v", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("事务提交失败: %v", err)
		}
		return nil
	})
	if err != nil {
		logging.Error("Failed to update order status", logrus.Fields{"error": err, "orderID": OrderID, "status": OrderStatus})
		return err
	}
	logging.Info("Order status updated successfully", logrus.Fields{"orderID": OrderID, "status": OrderStatus})
	return nil
}

// 查询订单状态
func QueryOrderStatus(db *sql.DB, orderID int) (*models.Order, error) {
	logging.Info("Querying order status", logrus.Fields{"orderID": orderID})
	var order models.Order
	err := monitoring.RecordDBTime("QueryOrderStatus", func() error {
		query := `SELECT orderid, riderid, shopid, productid, ordertime, totalprice, orderstatus FROM orders WHERE orderid = $1`
		row := db.QueryRow(query, orderID)
		err := row.Scan(&order.OrderID, &order.RiderID, &order.ShopID, &order.ProductID, &order.OrderTime, &order.TotalPrice, &order.OrderStatus)
		return err
	})

	if err != nil {
		logging.Error("Failed to query order status", logrus.Fields{"error": err, "orderID": orderID})
		return nil, err
	}
	return &order, nil
}

//骑手接单
func AcceptOrderTx(db *sql.DB, OrderID int, RiderID int) error {
	logging.Info("Accepting order", logrus.Fields{"orderID": OrderID, "riderID": RiderID})
	err := monitoring.RecordDBTime("AcceptOrderTx", func() error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()
		//检查订单状态
		var currentStatus string
		var currentRiderID sql.NullInt64                                          //不直接用int，是因为NULL 值会导致扫描错误，无法区分 0 和 NULL
		statusQuery := `SELECT orderstatus, riderid FROM orders WHERE orderid=$1 FOR UPDATE` //FOR UPDATA作用：行级锁定，事务隔离，避免并发问题
		err = tx.QueryRow(statusQuery, OrderID).Scan(&currentStatus, &currentRiderID)       //          优点：数据一致性，并发控制，业务安全
		if err != nil {
			return fmt.Errorf("获取订单出错：%v", err)
		}
		if currentStatus == "confirmed" { //已出餐

			return fmt.Errorf("订单已出餐，不允许接单")
		}
		if currentRiderID.Valid { //valid表示是否为NULL
			return fmt.Errorf("订单已被其他骑手接单")
		}
		//更新订单状态
		_, err = tx.Exec(`UPDATE orders SET riderid = $1,orderstatus = "delivering" WHERE orderid = $2 `, RiderID, OrderID)
		if err != nil {
			return fmt.Errorf("更新订单状态失败: %v", err)
		}

		//更新聊天群组，添加骑手
		_, err = tx.Exec(`UPDATE groups SET riderid = $1 WHERE orderid = $2`, RiderID, OrderID)
		if err != nil {
			return fmt.Errorf("更新聊天群组失败: %v", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交事务失败: %v", err)
		}
		return nil
	})
	if err != nil {
		logging.Error("Failed to accept order", logrus.Fields{"error": err, "orderID": OrderID, "riderID": RiderID})
		return err
	}
	logging.Info("Order accepted successfully", logrus.Fields{"orderID": OrderID, "riderID": RiderID})
	return nil
}

//完成订单
func CompleteOrderTx(db *sql.DB, OrderID int, RiderID int) error {
	logging.Info("Completing order", logrus.Fields{"orderID": OrderID, "riderID": RiderID})
	err := monitoring.RecordDBTime("CompleteOrderTx", func() error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()
		var currentStatus string
		statusQuery := `SELECT orderstatus FROM orders WHERE orderid=$1 FOR UPDATE`
		err = tx.QueryRow(statusQuery, OrderID).Scan(&currentStatus)
		if err != nil {
			return fmt.Errorf("查询订单状态失败：%v", err)
		}
		if currentStatus != "completed" {
			return fmt.Errorf("订单还未送达...")
		}

		//更新订单状态
		_, err = tx.Exec(`UPDATE orders SET orderstatus = "completed" WHERE orderid = $1`, OrderID)
		if err != nil {
			return fmt.Errorf("更新订单状态失败：%v", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交事务失败: %v", err)
		}
		return nil
	})
	if err != nil {
		logging.Error("Failed to complete order", logrus.Fields{"error": err, "orderID": OrderID, "riderID": RiderID})
		return err
	}
	logging.Info("Order completed successfully", logrus.Fields{"orderID": OrderID, "riderID": RiderID})
	return nil
}

// 取消订单
func DeleteOrder(db *sql.DB, OrderID int, ProductID int) error {
	logging.Info("Deleting order", logrus.Fields{"orderID": OrderID, "productID": ProductID})
	err := monitoring.RecordDBTime("DeleteOrder", func() error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()
		//检查订单状态
		var currentStatus string
		statusQuery := `SELECT orderstatus FROM orders WHERE orderid = $1 FOR UPDATE`
		err = tx.QueryRow(statusQuery, OrderID).Scan(&currentStatus)
		if err != nil {
			return fmt.Errorf("获取订单状态失败：%v", err)
		}

		if currentStatus == "completed" || currentStatus == "canceled" {
			return fmt.Errorf("订单已完成，无法取消")
		}

		//恢复商品库存
		_, err = tx.Exec(`UPDATE products SET stock = stock + 1 WHERE productid = (SELECT productid FROM orders WHERE orderid = $1)`, OrderID)
		if err != nil {
			return fmt.Errorf("恢复商品库存失败：%v", err)
		}

		//直接更新订购单状态
		_, err = tx.Exec(`UPDATE orders SET orderstatus = "canceled" WHERE orderid = $1`, OrderID)
		if err != nil {
			return fmt.Errorf("更新订单状态失败：%v", err)
		}
		return nil
	})
	if err != nil {
		logging.Error("Failed to delete order", logrus.Fields{"error": err, "orderID": OrderID, "productID": ProductID})
		return err
	}
	logging.Info("Order deleted successfully", logrus.Fields{"orderID": OrderID, "productID": ProductID})
	return nil
}
