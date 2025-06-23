package main

import "time"

// 用户结构体
type User struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
}

// 骑手的结构
type Rider struct {
	User                // 内嵌 User 结构体，继承用户的基本字段
	RiderID     int     `json:"rider_id"`     // 骑手ID，区分于 UserID
	VehicleType string  `json:"vehicle_type"` // 骑手使用的交通工具类型
	Rating      float64 `json:"rating"`       // 骑手的评分
	RiderStatus string  `json:"rider_status"` // 骑手状态（如在线、休息、离线）
	Latitude    float64 `json:"latitude"`     // 骑手的纬度
	Longitude   float64 `json:"longitude"`    // 骑手的经度
}

// 商家结构体
type Shop struct {
	ShopID       int     `json:"shop_id"`
	ShopName     string  `json:"shop_name"`
	ShopPassword string  `json:"shop_password"`
	Phone        string  `json:"phone"`
	Address      string  `json:"address"`
	Description  string  `json:"description"`
	Latitude     float64 `json:"latitude"`  // 商家的纬度
	Longitude    float64 `json:"longitude"` // 商家的经度
}

// 商品结构体
type Product struct {
	ProductID   int     `json:"product_id"`
	ShopID      int     `json:"shop_id"`
	ProductName string  `json:"product_name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// 订单结构体
type Order struct {
	OrderID     int       `json:"order_id"`
	UserID      int       `json:"user_id"`
	ShopID      int       `json:"shop_id"`
	RiderID     int       `json:"rider_id"`
	ProductID   int       `json:"product_id"`
	OrderStatus string    `json:"order_status"`
	Username    string    `json:"username"`
	ShopName    string    `json:"shop_name"`
	OrderTime   time.Time `json:"order_time"`
	ProductName string    `json:"product_name"`
	TotalPrice  float64   `json:"total_price"`
	GroupID     int       `json:"group_id,omitempty"`
}

// Group
type Group struct {
	GroupID int `json:"group_id,omitempty"`
	OrderID int `json:"order_id"`
	UserID  int `json:"user_id"`
	ShopID  int `json:"shop_id"`
	RiderID int `json:"rider_id"`
}

// Message 表示一条群聊消息的结构体
type Message struct {
	MessageID  int       `json:"message_id"`  // 某条消息的ID
	GroupID    int       `json:"group_id"`    // 群组ID
	SenderID   int       `json:"sender_id"`   // 发送者ID
	SenderName string    `json:"sender_name"` // "user", "shop", "rider"
	Content    string    `json:"content"`     // 消息内容
	Timestamp  time.Time `json:"timestamp"`   // 消息时间戳
}
