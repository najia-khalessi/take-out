package models

import "time"

type Review struct {
    ReviewID      int       `json:"review_id"`
    OrderID       int       `json:"order_id"`
    UserID        int       `json:"user_id"`
    ShopID        int       `json:"shop_id"`
    RiderID       *int      `json:"rider_id,omitempty"`
    Rating        int       `json:"rating"`
    Content       string    `json:"content"`
    SentimentScore float64  `json:"sentiment_score"`
    SentimentLabel string   `json:"sentiment_label"`
    ShopReply     *string   `json:"shop_reply,omitempty"`
    RepliedAt     *time.Time `json:"replied_at,omitempty"`
    IsAutoReview  bool      `json:"is_auto_review"`
    CreatedAt     time.Time `json:"created_at"`
}

type AIAnalysis struct {
    AnalysisID        int    `json:"analysis_id"`
    ReviewID          int    `json:"review_id"`
    EmotionalIntensity int   `json:"emotional_intensity"`

    DeliveryIssue      bool  `json:"delivery_issue"`
    FoodQualityIssue   bool  `json:"food_quality_issue"`
    ServiceIssue       bool  `json:"service_issue"`

    Summary20Chars     string `json:"summary_20chars"`
    SuggestedReply     string `json:"suggested_reply"`
}

// 商家评价查询请求
type ShopReviewsQuery struct {
    ShopID      int    `json:"shop_id"`
    StartDate   string `json:"start_date"`
    EndDate     string `json:"end_date"`
    Rating      *int   `json:"rating,omitempty"`
    Sentiment   *string `json:"sentiment,omitempty"`
    Page        int    `json:"page"`
    Size        int    `json:"size"`
}
