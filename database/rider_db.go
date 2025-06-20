package database
import (
	"context"
    "database/sql"
    "fmt"

    "take-out/models"
)
// 添加骑手
func insertRider(rp *RedisPool, db *sql.DB, rider *models.Rider) (int64, error) {
    // 插入用户基本信息
    userQuery := "INSERT INTO users (username, password, phone, address) VALUES (?, ?, ?, ?)"
    userResult, err := db.Exec(userQuery, rider.Username, rider.Password, rider.Phone, rider.Address)
    if err != nil {
        return 0, fmt.Errorf("插入用户信息失败: %v", err)
    }

    userID, err := userResult.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("获取用户ID失败: %v", err)
    }

    // 插入骑手特定信息
    riderQuery := `
        INSERT INTO riders (user_id, vehicle_type, rating, rider_status, latitude, longitude) 
        VALUES (?, ?, ?, ?, ?, ?)
    `
    result, err := db.Exec(riderQuery, 
        userID, 
        rider.VehicleType, 
        rider.Rating,
        rider.RiderStatus,
        rider.Latitude,
        rider.Longitude,
    )
    if err != nil {
        return 0, fmt.Errorf("插入骑手信息失败: %v", err)
    }

    riderID, err := result.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("获取骑手ID失败: %v", err)
    }

    // 存储到Redis
    ctx := context.Background()
    rdb := rp.GetClient()
    defer rp.PutClient(rdb)

    riderKey := fmt.Sprintf("rider:%d", riderID)
    err = rdb.HMSet(ctx, riderKey, map[string]interface{}{
        "user_id":      userID,
        "rider_id":     riderID,
        "username":     rider.Username,
        "phone":        rider.Phone,
        "address":      rider.Address,
        "vehicle_type": rider.VehicleType,
        "rating":       rider.Rating,
        "rider_status": rider.RiderStatus,
        "latitude":     rider.Latitude,
        "longitude":    rider.Longitude,
    }).Err()
    if err != nil {
        return 0, fmt.Errorf("存储到Redis失败: %v", err)
    }

    return riderID, nil
}

// 更新骑手信息
func UpdateRider(db *sql.DB,OrderID int, RiderID int) error {
    // 开始一个事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("无法开始事务: %v", err)
	}
	defer tx.Rollback()
	// 执行更新订单状态的 SQL 查询
	query := "UPDATE orders SET rider_id = ? WHERE order_id = ?"
	_, err = tx.Exec(query, RiderID, OrderID)
	if err != nil {
		return fmt.Errorf("订单状态更新失败: %v", err)
	}
	if err := tx.Commit(); err != nil {
        return 0, fmt.Errorf("事务提交失败: %v", err)
    }

	return nil
}

//订阅频道
// 1. 订阅专属频道
func riderSubscribe(riderID int) {
    pubsub := rdb.Subscribe(ctx, fmt.Sprintf("rider_%d", riderID))
    for msg := range pubsub.Channel() {
        var order map[string]interface{}
        json.Unmarshal([]byte(msg.Payload), &order)
        
        // 弹窗提示骑手
        ShowOrderPopup(order) 
    }
}
// 2. 用户点击接单后调用
func onAcceptOrder(orderID int) {
    http.Post("/rider/response", JSON{
        "order_id": orderID,
        "action":   "accept"
    })
}