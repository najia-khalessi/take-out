package database

import (
	"database/sql"
	"take-out/models"
)

// InsertReview 将评价插入数据库
func InsertReview(db *sql.DB, review *models.Review) (int, error) {
	query := `
		INSERT INTO reviews (order_id, user_id, shop_id, rider_id, rating, content, is_auto_review)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING review_id
	`
	var reviewID int
	err := db.QueryRow(
		query,
		review.OrderID,
		review.UserID,
		review.ShopID,
		review.RiderID,
		review.Rating,
		review.Content,
		review.IsAutoReview,
	).Scan(&reviewID)

	if err != nil {
		return 0, err
	}

	return reviewID, nil
}
