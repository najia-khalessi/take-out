package database

import (
	"take-out/models"
	"database/sql"
	"fmt"
)
// CreateGroup 创建与订单关联的新聊天群组
func CreateGroup(order map[string]interface{}, rp *RedisPool, db *sql.DB, tx *sql.Tx) error {
	group := &Group{
		OrderID: int(order["order_id"].(float64)), // JSON 反序列化时数字会被转换为 float64
		UserID:  int(order["user_id"].(float64)),
		ShopID:  int(order["shop_id"].(float64)),
		RiderID: int(order["rider_id"].(float64)),
	}

	groupID, err := insertGroup(tx, rp, group) // 使用传入的事务对象
	if err != nil {
		return fmt.Errorf("创建群组失败: %v", err)
	}

	fmt.Printf("群组创建成功, ID: %d\n", groupID)
	return nil
}
// insertGroup 插入群组信息到数据库和Redis，使用传入的事务对象
func insertGroup(tx *sql.Tx, rp *RedisPool, group *Group) (int64, error) {
    // 使用提供的事务对象将群组插入MySQL
    query := "INSERT INTO `groups` (order_id, user_id, shop_id, rider_id) VALUES (?, ?, ?, ?)"
    result, err := tx.Exec(query, group.OrderID, group.UserID, group.ShopID, group.RiderID)
    if err != nil {
        return 0, fmt.Errorf("MySQL群组插入失败: %v", err)
    }
    groupID, err := result.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("获取MySQL群组ID失败: %v", err)
    }

    // 将群组信息插入Redis
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
    if err != nil {
        return 0, fmt.Errorf("Redis群组插入失败: %v", err)
    }

    return groupID, nil
}