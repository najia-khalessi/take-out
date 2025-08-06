package database

import (
	"context"
	"database/sql"
	"fmt"
	"take-out/logging"
	"take-out/models"
	"take-out/monitoring"
	"time"

	"github.com/sirupsen/logrus"
)

/* IM即时通讯系统
借助redis的MQ实现Rider抢单和派单系统，按照某个策略丢到公共大厅供大家抢，可以做成随机的策略。
User点单后生成OrderlD，Shop拿到OrderlD，把OrderlD发布到MQ，Rider通过消费MQ(Rider拿到OrderlD)，
IM系统拿到OrderlD后读MQ(Order结构体下有UserlD，ShoplD，RiderlD)
同时新增一个GrouplD(Group结构体下有ShopIDUserlD，RiderlD，OrderD)，
创建了一个群聊，实现聊天,groupid应该是和orderid绑定的。
*/

// NewGroupMessage 创建带有时间戳的新消息
func NewGroupMessage(messageID, groupID, senderID int, senderName, content string) *models.Message {
	return &models.Message{
		MessageID:  messageID,
		GroupID:    groupID,
		SenderID:   senderID,
		SenderName: senderName,
		Content:    content,
		Timestamp:  time.Now(),
	}
}

// PublishMessage 将消息发布到 Redis 频道
func PublishMessage(rp *RedisPool, channel, message string) error {
	logging.Info("Publishing message", logrus.Fields{"channel": channel})
	err := monitoring.RecordRedisTime("PublishMessage", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rdb := rp.GetClient()
		defer rp.PutClient(rdb)

		_, err := rdb.Publish(ctx, channel, message).Result()
		return err
	})
	if err != nil {
		logging.Error("Failed to publish message to Redis", logrus.Fields{"error": err, "channel": channel})
		return fmt.Errorf("发布消息到 Redis 失败: %v", err)
	}
	return nil
}

// SaveMessage 将消息保存到数据库中，确保按时间戳排序
func SaveMessage(db *sql.DB, msg *models.Message) error {
	logging.Info("Saving message", logrus.Fields{"groupID": msg.GroupID, "senderID": msg.SenderID})
	err := monitoring.RecordDBTime("SaveMessage", func() error {
		query := "INSERT INTO messages (groupid, sender_id, sender_name, content, timestamp) VALUES ($1, $2, $3, $4, $5)"
		_, err := db.Exec(query, msg.GroupID, msg.SenderID, msg.SenderName, msg.Content, msg.Timestamp)
		return err
	})
	if err != nil {
		logging.Error("Failed to save message", logrus.Fields{"error": err, "groupID": msg.GroupID, "senderID": msg.SenderID})
		return fmt.Errorf("保存消息失败: %v", err)
	}
	return nil
}

// CleanUpOldRecords 清理旧的订单和消息记录
func CleanUpOldRecords(db *sql.DB) error {
	logging.Info("Cleaning up old records", nil)
	err := monitoring.RecordDBTime("CleanUpOldRecords.Orders", func() error {
		_, err := db.Exec("DELETE FROM orders WHERE ordertime < NOW() - INTERVAL '2 weeks'")
		return err
	})
	if err != nil {
		logging.Error("Failed to clean up old orders", logrus.Fields{"error": err})
		return fmt.Errorf("清理订单记录失败: %v", err)
	}

	err = monitoring.RecordDBTime("CleanUpOldRecords.Messages", func() error {
		_, err := db.Exec("DELETE FROM messages WHERE timestamp < NOW() - INTERVAL '2 weeks'")
		return err
	})
	if err != nil {
		logging.Error("Failed to clean up old messages", logrus.Fields{"error": err})
		return fmt.Errorf("清理消息记录失败: %v", err)
	}

	logging.Info("Old records cleaned up successfully", nil)
	return nil
}

// StartWeeklyCleanUpScheduler 启动每周清理调度器
func StartWeeklyCleanUpScheduler(db *sql.DB) {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C { // 使用 for range 循环来处理通道接收
		logging.Info("Starting weekly cleanup task", nil)
		err := CleanUpOldRecords(db)
		if err != nil {
			logging.Error("Weekly cleanup task failed", logrus.Fields{"error": err})
		} else {
			logging.Info("Weekly cleanup task completed successfully", nil)
		}
	}
}