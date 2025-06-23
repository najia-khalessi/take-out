package database

import (
	"take-out/models"
	"database/sql"
	"fmt"
)
/* IM即时通讯系统
借助redis的MQ实现Rider抢单和派单系统，按照某个策略丢到公共大厅供大家抢，可以做成随机的策略。
User点单后生成OrderlD，Shop拿到OrderlD，把OrderlD发布到MQ，Rider通过消费MQ(Rider拿到OrderlD)，
IM系统拿到OrderlD后读MQ(Order结构体下有UserlD，ShoplD，RiderlD)
同时新增一个GrouplD(Group结构体下有ShopIDUserlD，RiderlD，OrderD)，
创建了一个群聊，实现聊天,groupid应该是和orderid绑定的。
*/

// NewGroupMessage 创建带有时间戳的新消息
func NewGroupMessage(messageID, groupID, senderID int, content string) *Message {
	return &Message{
		MessageID: messageID,
		GroupID:   groupID,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// PublishMessage 将消息发布到 Redis 频道
func PublishMessage(rp *RedisPool, channel, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rdb := rp.GetClient()
	defer rp.PutClient(rdb)

	_, err := rdb.Publish(ctx, channel, message).Result()
	if err != nil {
		return fmt.Errorf("发布消息到 Redis 失败: %v", err)
	}
	return nil
}

// SaveMessage 将消息保存到数据库中，确保按时间戳排序
func SaveMessage(db *sql.DB, msg *Message) error {
	query := "INSERT INTO messages (group_id, sender_id, sender_name, content, timestamp) VALUES (?, ?, ?, ?, ?)"
	_, err := db.Exec(query, msg.GroupID, msg.SenderID, msg.SenderName, msg.Content, msg.Timestamp)
	if err != nil {
		return fmt.Errorf("保存消息失败: %v", err)
	}
	return nil
}

// CleanUpOldRecords 清理旧的订单和消息记录
func CleanUpOldRecords(db *sql.DB) error {
	// 删除两周前的订单记录
	_, err := db.Exec("DELETE FROM orders WHERE YEARWEEK(order_time, 1) = YEARWEEK(NOW() - INTERVAL 2 WEEK, 1)")
	if err != nil {
		return fmt.Errorf("清理订单记录失败: %v", err)
	}

	// 删除两周前的消息记录
	_, err = db.Exec("DELETE FROM messages WHERE YEARWEEK(timestamp, 1) = YEARWEEK(NOW() - INTERVAL 2 WEEK, 1)")
	if err != nil {
		return fmt.Errorf("清理消息记录失败: %v", err)
	}

	log.Println("旧记录清理成功")
	return nil
}

// StartWeeklyCleanUpScheduler 启动每周清理调度器
func StartWeeklyCleanUpScheduler(db *sql.DB) {
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C { // 使用 for range 循环来处理通道接收
		log.Println("开始每周清理任务...")
		err := CleanUpOldRecords(db)
		if err != nil {
			log.Printf("每周清理任务失败: %v", err)
		} else {
			log.Println("每周清理任务成功完成")
		}
	}
}
