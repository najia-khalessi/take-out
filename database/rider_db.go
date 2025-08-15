package database

import (
	"database/sql"
	"fmt"
	"take-out/logging"
	"take-out/models"
	"take-out/monitoring"

	"github.com/sirupsen/logrus"
)

// InsertRider adds a new rider to the database
func InsertRider(db *sql.DB, rider *models.Rider) (int64, error) {
	logging.Info("InsertRider called", logrus.Fields{"ridername": rider.RiderName})
	var riderID int64
	err := monitoring.RecordDBTime("InsertRider", func() error {
		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			logging.Error("Failed to begin transaction", logrus.Fields{"error": err})
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()

		// Check if ridername already exists
		var count int
		checkQuery := `SELECT COUNT(*) FROM riders WHERE ridername = $1`
		err = tx.QueryRow(checkQuery, rider.RiderName).Scan(&count)
		if err != nil {
			logging.Error("Failed to query ridername", logrus.Fields{"error": err})
			return fmt.Errorf("查询骑手用户名失败: %v", err)
		}
		if count > 0 {
			logging.Warn("Ridername already exists", logrus.Fields{"ridername": rider.RiderName})
			return fmt.Errorf("骑手用户名已存在")
		}

		// Insert into database
		query := "INSERT INTO riders (ridername, riderpassword, riderphone, vehicletype) VALUES ($1, $2, $3, $4) RETURNING riderid"
		err = tx.QueryRow(query, rider.RiderName, rider.RiderPassword, rider.RiderPhone, rider.VehicleType).Scan(&riderID)
		if err != nil {
			logging.Error("Failed to insert rider into database", logrus.Fields{"error": err})
			return fmt.Errorf("插入数据库失败: %v", err)
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			logging.Error("Failed to commit transaction", logrus.Fields{"error": err})
			return fmt.Errorf("提交事务失败: %v", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	logging.Info("Successfully inserted new rider", logrus.Fields{"riderID": riderID})
	return riderID, nil
}