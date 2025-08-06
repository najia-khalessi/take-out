package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"take-out/database"
	"take-out/models"

	"github.com/redis/go-redis/v9"
)

// HandleSendMessage 处理通过 HTTP 请求发送消息
// 先将消息保存到数据库，然后再发布到 Redis
// 如果数据库保存失败，可以立即返回错误，而不必处理已发送的 Redis 消息可能带来的不一致问题
// 避免了在发布消息到 Redis 后数据库写入失败的情况
func HandleSendMessage(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg models.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			log.Printf("解析请求体失败: %v", err)
			http.Error(w, "无效的请求体", http.StatusBadRequest)
			return
		}

		// 设置当前时间戳
		msg.Timestamp = time.Now()

		// 将消息先保存到数据库，确保数据持久化
		if err := database.SaveMessage(db, &msg); err != nil {
			log.Printf("保存消息失败: %v", err)
			http.Error(w, fmt.Sprintf("保存消息失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 将消息发布到 Redis 频道，供其他组件实时处理
		channel := fmt.Sprintf("group_%d", msg.GroupID)
		if err := database.PublishMessage(rp, channel, msg.Content); err != nil {
			log.Printf("发布消息失败: %v", err)
			http.Error(w, fmt.Sprintf("发布消息失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 返回成功响应
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "消息发送成功"})
	}
}

// HandleGetMessages 处理获取特定群组消息的 HTTP 请求
func HandleGetMessages(db *sql.DB, rp *database.RedisPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID := r.URL.Query().Get("group_id")
		if groupID == "" {
			http.Error(w, "缺少 group_id 参数", http.StatusBadRequest)
			return
		}

		rdb := rp.GetClient()
		defer rp.PutClient(rdb)

		// 从 Redis 缓存中获取消息
		cachedMessages, err := rdb.Get(context.Background(), groupID).Result()
		if err == redis.Nil {
			// 从数据库中按时间戳顺序获取消息
			rows, err := db.Query("SELECT group_id, sender_id, sender_name, content, timestamp FROM messages WHERE group_id = ? ORDER BY timestamp ASC", groupID)
			if err != nil {
				http.Error(w, fmt.Sprintf("查询消息失败: %v", err), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var messages []models.Message
			for rows.Next() {
				var msg models.Message
				if err := rows.Scan(&msg.GroupID, &msg.SenderID, &msg.SenderName, &msg.Content, &msg.Timestamp); err != nil {
					http.Error(w, fmt.Sprintf("解析消息失败: %v", err), http.StatusInternalServerError)
					return
				}
				messages = append(messages, msg)
			}

			messagesJSON, _ := json.Marshal(messages)
			rdb.Set(context.Background(), groupID, messagesJSON, 7*24*time.Hour) // 缓存消息 7 天

			w.Header().Set("Content-Type", "application/json")
			w.Write(messagesJSON)
		} else if err != nil {
			http.Error(w, fmt.Sprintf("Redis 查询失败: %v", err), http.StatusInternalServerError)
			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cachedMessages))
		}
	}
}