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
	"golang.org/x/crypto/bcrypt"
)

//都使用事务，确保数据一致性+出错时需要回滚

//创建用户
func InsertUser(rp *RedisPool, db *sql.DB, user *models.User) (int64, error) {
	logging.Info("Attempting to insert a new user", logrus.Fields{"username": user.Username})
	var userID int64
	err := monitoring.RecordDBTime("InsertUser", func() error {
		//开启事务
		tx, err := db.Begin()
		if err != nil {
			logging.Error("Failed to begin transaction", logrus.Fields{"error": err})
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()

		//检查用户名是否已存在，确保用户名的唯一性
		var count int
		checkQuery := `SELECT COUNT(*) FROM users WHERE username = $1`
		err = tx.QueryRow(checkQuery, user.Username).Scan(&count)
		if err != nil {
			logging.Error("Failed to query username", logrus.Fields{"error": err})
			return fmt.Errorf("查询用户名失败: %v", err)
		}
		if count > 0 {
			logging.Warn("Username already exists", logrus.Fields{"username": user.Username})
			return fmt.Errorf("用户名已存在")
		}

		//插入数据库
		query := "INSERT INTO users (username, userpassword, userphone, useraddress) VALUES ($1, $2, $3, $4) RETURNING userid"
		err = tx.QueryRow(query, user.Username, user.UserPassword, user.UserPhone, user.UserAddress).Scan(&userID)
		if err != nil {
			logging.Error("Failed to insert user into database", logrus.Fields{"error": err})
			return fmt.Errorf("插入数据库失败: %v", err)
		}

		//提交事务
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
	// 插入Redis
	ctx := context.Background()
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	err = rdb.HMSet(ctx, fmt.Sprintf("user:%d", userID), map[string]interface{}{
		"user_id":       userID,
		"username":      user.Username,
		"user_password": user.UserPassword,
		"phone":         user.UserPhone,
		"address":       user.UserAddress,
	}).Err()
	if err != nil {
		logging.Warn("Failed to insert user into Redis", logrus.Fields{"error": err})
	}

	logging.Info("Successfully inserted new user", logrus.Fields{"userID": userID})
	return userID, nil
}

// 更新用户
func UpdateUser(rp *RedisPool, db *sql.DB, user *models.User) error {
	logging.Info("Attempting to update user", logrus.Fields{"userID": user.UserID})
	err := monitoring.RecordDBTime("UpdateUser", func() error {
		// 开启事务
		tx, err := db.Begin()
		if err != nil {
			logging.Error("Failed to begin transaction", logrus.Fields{"error": err})
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()

		// 确保用户存在
		var count int
		checkQuery := `SELECT COUNT(*) FROM users WHERE userid = $1`
		err = tx.QueryRow(checkQuery, user.UserID).Scan(&count)
		if err != nil {
			logging.Error("Failed to query user", logrus.Fields{"error": err})
			return fmt.Errorf("查询用户失败: %v", err)
		}
		if count == 0 {
			logging.Warn("User does not exist", logrus.Fields{"userID": user.UserID})
			return fmt.Errorf("用户不存在")
		}

		// 更新数据库
		updateQuery := `UPDATE users 
                   SET username = $1, 
                       userpassword = $2, 
                       userphone = $3, 
                       useraddress = $4 
                   WHERE userid = $5`
		_, err = tx.Exec(updateQuery,
			user.Username,
			user.UserPassword,
			user.UserPhone,
			user.UserAddress,
			user.UserID)
		if err != nil {
			logging.Error("Failed to update user in database", logrus.Fields{"error": err})
			return fmt.Errorf("更新数据库失败: %v", err)
		}

		// 提交事务
		if err = tx.Commit(); err != nil {
			logging.Error("Failed to commit transaction", logrus.Fields{"error": err})
			return fmt.Errorf("提交事务失败: %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 更新Redis缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() //创建一个超时控制上下文对象
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)

	err = rdb.HMSet(ctx,
		fmt.Sprintf("user:%d", user.UserID), //使用user_id作为Redis的key
		map[string]interface{}{ //哈希表内容,key-value对
			"user_id":       user.UserID,
			"username":      user.Username,
			"user_password": user.UserPassword,
			"phone":         user.UserPhone,
			"address":       user.UserAddress,
		}).Err()
	if err != nil {
		// Redis更新失败不影响主流程
		logging.Warn("Failed to update user in Redis", logrus.Fields{"error": err})
	}

	logging.Info("Successfully updated user", logrus.Fields{"userID": user.UserID})
	return nil
}

//获取所有用户(分页查询用户)，为了查看所有用户，监控用户状态，分析用户分布等等需要所有用户的数据时候用得到
func GetAllUsers(db *sql.DB, offset, limit int) ([]models.User, error) {
	logging.Info("Attempting to get all users", logrus.Fields{"offset": offset, "limit": limit})
	var users []models.User //因为要返回多个用户数据，所以设计成User类型的切片
	query := `SELECT userid, username, userphone, useraddress
			FROM users 
			LIMIT $1 OFFSET $2`
	//OFFSET = (页码 - 1) * 每页数量   LIMIT = 每页数量

	var rows *sql.Rows
	var err error
	err = monitoring.RecordDBTime("GetAllUsers", func() error {
		rows, err = db.Query(query, limit, offset)
		return err
	})
	if err != nil {
		logging.Error("Failed to get all users", logrus.Fields{"error": err})
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User                                                              // 创建单个用户变量
		err := rows.Scan(&user.UserID, &user.Username, &user.UserPhone, &user.UserAddress) // 填充用户数据
		if err != nil {
			logging.Error("Failed to scan user row", logrus.Fields{"error": err})
			return nil, err
		}
		users = append(users, user) // 将用户添加到切片中，append可以自动扩容
	}
	logging.Info("Successfully retrieved all users", logrus.Fields{"count": len(users)})
	return users, nil
}

//删除用户
func DeleteUser(rp *RedisPool, db *sql.DB, userID int) error {
	logging.Info("Attempting to delete user", logrus.Fields{"userID": userID})
	err := monitoring.RecordDBTime("DeleteUser", func() error {
		// 开启事务
		tx, err := db.Begin()
		if err != nil {
			logging.Error("Failed to begin transaction", logrus.Fields{"error": err})
			return fmt.Errorf("开启事务失败: %v", err)
		}
		defer tx.Rollback()

		// 删除数据库中的用户
		deleteQuery := `DELETE FROM users WHERE userid = $1`
		_, err = tx.Exec(deleteQuery, userID)
		if err != nil {
			logging.Error("Failed to delete user from database", logrus.Fields{"error": err})
			return fmt.Errorf("删除数据库用户失败: %v", err)
		}

		// 提交事务
		if err = tx.Commit(); err != nil {
			logging.Error("Failed to commit transaction", logrus.Fields{"error": err})
			return fmt.Errorf("提交事务失败: %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 从Redis中删除用户缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)

	err = rdb.Del(ctx, fmt.Sprintf("user:%d", userID)).Err()
	if err != nil {
		logging.Warn("Failed to delete user from Redis", logrus.Fields{"error": err})
	}

	logging.Info("Successfully deleted user", logrus.Fields{"userID": userID})
	return nil
}

// ValidateUser 验证用户凭据
func ValidateUser(db *sql.DB, userphone, password string) (*models.User, error) {
	logging.Info("Attempting to validate user", logrus.Fields{"userphone": userphone})
	var user models.User
	query := "SELECT userid, username, userpassword, userphone, useraddress FROM users WHERE userphone = $1"
	err := monitoring.RecordDBTime("ValidateUser", func() error {
		return db.QueryRow(query, userphone).Scan(&user.UserID, &user.Username, &user.UserPassword, &user.UserPhone, &user.UserAddress)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Warn("User not found", logrus.Fields{"userphone": userphone})
			return nil, fmt.Errorf("用户不存在")
		}
		logging.Error("Failed to query user", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.UserPassword), []byte(password))
	if err != nil {
		logging.Warn("Incorrect password", logrus.Fields{"userphone": userphone})
		return nil, fmt.Errorf("密码错误")
	}
	logging.Info("Successfully validated user", logrus.Fields{"userID": user.UserID})
	return &user, nil
}


// GetUserByID 从数据库获取用户信息
func GetUserByID(db *sql.DB, userID int) (*models.User, error) {
	logging.Info("Attempting to get user by ID", logrus.Fields{"userID": userID})
	var user models.User
	query := "SELECT userid, username, userphone, useraddress FROM users WHERE userid = $1"
	err := monitoring.RecordDBTime("GetUserByID", func() error {
		return db.QueryRow(query, userID).Scan(&user.UserID, &user.Username, &user.UserPhone, &user.UserAddress)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Warn("User not found", logrus.Fields{"userID": userID})
			return nil, fmt.Errorf("用户不存在")
		}
		logging.Error("Failed to query user by ID", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}
	logging.Info("Successfully retrieved user by ID", logrus.Fields{"userID": user.UserID})
	return &user, nil
}

// GetUserByUsername 从数据库获取用户信息
func GetUserByUsername(db *sql.DB, username string) (*models.User, error) {
	logging.Info("Attempting to get user by username", logrus.Fields{"username": username})
	var user models.User
	query := "SELECT userid, username, userphone, useraddress FROM users WHERE username = $1"
	err := monitoring.RecordDBTime("GetUserByUsername", func() error {
		return db.QueryRow(query, username).Scan(&user.UserID, &user.Username, &user.UserPhone, &user.UserAddress)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Warn("User not found", logrus.Fields{"username": username})
			return nil, fmt.Errorf("用户不存在")
		}
		logging.Error("Failed to query user by username", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}
	logging.Info("Successfully retrieved user by username", logrus.Fields{"userID": user.UserID})
	return &user, nil
}

// ValidateRider 验证骑手凭据
func ValidateRider(db *sql.DB, riderphone, password string) (*models.Rider, error) {
	logging.Info("Attempting to validate rider", logrus.Fields{"riderphone": riderphone})
	var rider models.Rider
	query := "SELECT riderid, ridername, riderpassword, riderphone, vehicletype FROM riders WHERE riderphone = $1"
	err := monitoring.RecordDBTime("ValidateRider", func() error {
		return db.QueryRow(query, riderphone).Scan(&rider.RiderID, &rider.RiderName, &rider.RiderPassword, &rider.RiderPhone, &rider.VehicleType)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Warn("Rider not found", logrus.Fields{"riderphone": riderphone})
			return nil, fmt.Errorf("骑手不存在")
		}
		logging.Error("Failed to query rider", logrus.Fields{"error": err})
		return nil, fmt.Errorf("查询骑手失败: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(rider.RiderPassword), []byte(password))
	if err != nil {
		logging.Warn("Incorrect password", logrus.Fields{"riderphone": riderphone})
		return nil, fmt.Errorf("密码错误")
	}
	logging.Info("Successfully validated rider", logrus.Fields{"riderID": rider.RiderID})
	return &rider, nil
}
