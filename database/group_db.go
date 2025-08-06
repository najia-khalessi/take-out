package database

import (
	"context"
	"database/sql"
	"fmt"
	"take-out/logging"
	"take-out/models"
	"take-out/monitoring"

	"github.com/sirupsen/logrus"
)

// CreateGroup 创建与订单关联的新聊天群组
func CreateGroup(order map[string]interface{}, rp *RedisPool, db *sql.DB, tx *sql.Tx) error {
	logging.Info("Creating group", logrus.Fields{"order": order})
	group := &models.Group{
		OrderID: int(order["order_id"].(float64)), // JSON 反序列化时数字会被转换为 float64
		UserID:  int(order["user_id"].(float64)),
		ShopID:  int(order["shop_id"].(float64)),
		RiderID: int(order["rider_id"].(float64)),
	}

	groupID, err := InsertGroup(tx, rp, group) // 使用传入的事务对象
	if err != nil {
		logging.Error("Failed to create group", logrus.Fields{"error": err})
		return fmt.Errorf("创建群组失败: %v", err)
	}

	logging.Info("Group created successfully", logrus.Fields{"groupID": groupID})
	return nil
}

// InsertGroup 插入群组信息到数据库和Redis，使用传入的事务对象
func InsertGroup(tx *sql.Tx, rp *RedisPool, group *models.Group) (int64, error) {
	var groupID int64
	err := monitoring.RecordDBTime("InsertGroup", func() error {
		// 使用提供的事务对象将群组插入PostgreSQL
		query := "INSERT INTO `groups` (orderid, userid, shopid, riderid) VALUES ($1, $2, $3, $4) RETURNING groupid"
		err := tx.QueryRow(query, group.OrderID, group.UserID, group.ShopID, group.RiderID).Scan(&groupID)
		if err != nil {
			return fmt.Errorf("PostgreSQL群组插入失败: %v", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	// 将群组信息插入Redis
	err = monitoring.RecordRedisTime("InsertGroup", func() error {
		ctx := context.Background()
		rdb := rp.GetClient()
		defer rp.PutClient(rdb)
		err = rdb.HMSet(ctx, fmt.Sprintf("group:%d", groupID), map[string]interface{}{
			"group_id": groupID,
			"order_id": group.OrderID,
			"user_id":  group.UserID,
			"shop_id":  group.ShopID,
			"rider_id": group.RiderID,
		}).Err()
		return err
	})
	if err != nil {
		// Redis 失败不应导致整个事务失败，但应记录警告
		logging.Warn("Failed to insert group into Redis", logrus.Fields{"error": err, "groupID": groupID})
	}

	return groupID, nil
}
