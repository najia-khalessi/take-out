package database

import (
	"take-out/models"
	"database/sql"
	"fmt"
)

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