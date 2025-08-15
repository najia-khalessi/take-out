package models

import (
	"time"
	"github.com/golang-jwt/jwt"
)

// 用户结构体
type User struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"user_name"`
	UserPassword string `json:"user_password"`
	UserPhone    string `json:"user_phone"`
	UserAddress  string `json:"user_address"`
}

// 骑手的结构
type Rider struct {
	RiderID          int     `json:"rider_id"`      // 骑手ID，区分于 UserID
	RiderName        string  `json:"rider_name"`    // 骑手名
	RiderPassword    string  `json:"rider_password"`// 骑手密码
	RiderRating      float64 `json:"rider_rating"`  // 骑手的评分
	RiderPhone       string  `json:"rider_phone"`
	VehicleType      string  `json:"vehicle_type"`   // 车辆类型
	RiderStatus      string  `json:"rider_status"`   // 骑手状态（如在线、休息、离线）
	RiderLatitude    float64 `json:"rider_latitude"` // 骑手的纬度
	RiderLongitude   float64 `json:"rider_longitude"`// 骑手的经度
	DeliveryFee      float64 `json:"delivery_fee"`   // 配送费
}

// 商家结构体
type Shop struct {
	ShopID       int     `json:"shop_id"`
	ShopName     string  `json:"shop_name"`
	ShopPassword string  `json:"shop_password"`
	ShopPhone        string  `json:"shop_phone"`
	ShopAddress      string  `json:"shop_address"`
	Description  string  `json:"description"`
	ShopLatitude     float64 `json:"shop_latitude"`  // 商家的纬度
	ShopLongitude    float64 `json:"shop_longitude"` // 商家的经度
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
	DeliveryFee float64   `json:"delivery_fee"` // 配送费
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

// TokenPair包含访问令牌和刷新令牌
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Claims 是JWT中的声明
type Claims struct {
	UserID int    `json:"user_id,omitempty"`
	ShopID int    `json:"shop_id,omitempty"`
	RiderID int    `json:"rider_id,omitempty"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.StandardClaims
}

// TokenBlacklist for database
type TokenBlacklist struct {
	JTI       string    `gorm:"primary_key"`
	ExpiresAt time.Time `gorm:"not null"`
}