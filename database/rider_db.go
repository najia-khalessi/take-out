package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"take-out/models"
	"time"
)

type FeeRule struct {
	StartHour int     // 起始小时（24小时制）
	EndHour   int     // 结束小时
	BaseFee   float64 // 起步费
}

var feeRules = []FeeRule{
	{StartHour: 8, EndHour: 20, BaseFee: 5.0},  // 白天
	{StartHour: 20, EndHour: 24, BaseFee: 8.0}, // 夜间
	{StartHour: 0, EndHour: 8, BaseFee: 8.0},   // 凌晨
}

// 根据当前时间获取起步费
func GetBaseFeeByTime(now time.Time) float64 {
	hour := now.Hour()
	for _, rule := range feeRules {
		if rule.StartHour <= hour && hour < rule.EndHour {
			return rule.BaseFee
		}
		// 跨天时间段
		if rule.StartHour > rule.EndHour && (hour >= rule.StartHour || hour < rule.EndHour) {
			return rule.BaseFee
		}
	}
	return 5.0 // 默认
}

// 添加骑手
// 注意: 此函数假定 'riders' 表有一个 'user_id' 列来链接到 'users' 表。
// 当前的 init.sql 文件没有体现这一点。
func InsertRider(rp *RedisPool, db *sql.DB, rider *models.Rider) (int64, error) {
	// 插入骑手特定信息
	riderQuery := `
        INSERT INTO riders (user_id, vehicle_type, rider_rating, rider_status, rider_latitude, rider_longitude, delivery_fee) 
        VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING riderid
    `
	var riderID int64
	err := db.QueryRow(riderQuery,
		rider.UserID,
		rider.VehicleType,
		rider.RiderRating,
		rider.RiderStatus,
		rider.RiderLatitude,
		rider.RiderLongitude,
		rider.DeliveryFee,
	).Scan(&riderID)
	if err != nil {
		return 0, fmt.Errorf("插入骑手信息失败: %v", err)
	}

	// 存储到Redis
	ctx := context.Background()
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)

	riderKey := fmt.Sprintf("rider:%d", riderID)
	err = rdb.HMSet(ctx, riderKey, map[string]interface{}{
		"user_id":         rider.UserID,
		"rider_id":        riderID,
		"vehicle_type":    rider.VehicleType,
		"rider_rating":    rider.RiderRating,
		"rider_status":    rider.RiderStatus,
		"rider_latitude":  rider.RiderLatitude,
		"rider_longitude": rider.RiderLongitude,
		"delivery_fee":    rider.DeliveryFee,
	}).Err()
	if err != nil {
		log.Printf("警告: 存储骑手信息到Redis失败: %v", err)
	}

	return riderID, nil
}

// AssignRiderToOrder 更新订单的骑手信息
func AssignRiderToOrder(db *sql.DB, orderID int, riderID int) error {
	// 开始一个事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("无法开始事务: %v", err)
	}
	defer tx.Rollback()
	// 执行更新订单状态的 SQL 查询
	query := "UPDATE orders SET rider_id = $1 WHERE order_id = $2"
	_, err = tx.Exec(query, riderID, orderID)
	if err != nil {
		return fmt.Errorf("订单状态更新失败: %v", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("事务提交失败: %v", err)
	}

	return nil
}

/*
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
*/

// 计算两点间距离（单位：公里）
func CalcDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // 地球半径，单位km
	latRad1 := lat1 * math.Pi / 180
	latRad2 := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(latRad1)*math.Cos(latRad2)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
