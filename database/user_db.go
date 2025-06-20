package database

import (
	"take-out/models"
	"database/sql"
	"fmt"
)
//都使用事务，确保数据一致性+出错时需要回滚

//创建用户
func insertUser(rp *RedisPool, db *sql.DB, user *models.User) (int64, error) {
	//开启事务
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启事务失败: %v", err)
	}
	defer tx.Rollback()

	//检查用户名是否已存在，确保用户名的唯一性
	var count int
	checkQuery := `SELECT COUNT(*) FROM users WHERE username = ?`
	err := tx.QueryRow(checkQuery, user.UserName).Scan(&count)
	if err != nil {
		return fmt.Errorf("查询用户名失败: %v", err)
	}
	if count > 0 {
	   return fmt.Errorf("用户名已存在")
	}

	//插入数据库
	query := "INSERT INTO users (username, password, phone, address) VALUES (?, ?, ?, ?)"
	result, err := db.Exec(query, user.Username, user.Password, user.Phone, user.Address)
	if err != nil {
		return 0, fmt.Errorf("插入数据库失败: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取用户ID失败: %v", err)
	}
	//提交事务
	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("提交事务失败: %v", err)
	}
	// 插入Redis
	ctx := context.Background()
	rdb := rp.GetClient()
	defer rp.PutClient(rdb)
	err = rdb.HMSet(ctx, fmt.Sprintf("user:%d", userID), map[string]interface{}{
		"user_id":  userID,
		"username": user.Username,
		"password": user.Password,
		"phone":    user.Phone,
		"address":  user.Address,
	}).Err()
	if err != nil {
		return 0, fmt.Errorf("警告: 插入Redis失败: %v", err)
	}

	return userID, nil
}


// 更新用户
func UpdateUser(rp *RedisPool, db *sql.DB, user *models.User) error {
    // 开启事务
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("开启事务失败: %v", err)
    }
    defer tx.Rollback()

    // 确保用户存在
    var count int
    checkQuery := `SELECT COUNT(*) FROM users WHERE user_id = ?`
    err = tx.QueryRow(checkQuery, user.UserID).Scan(&count)
    if err != nil {
        return fmt.Errorf("查询用户失败: %v", err)
    }
    if count == 0 {
        return fmt.Errorf("用户不存在")
    }

    // 更新数据库
    updateQuery := `UPDATE users 
                   SET username = ?, 
                       password = ?, 
                       phone = ?, 
                       address = ? 
                   WHERE user_id = ?`
    _, err = tx.Exec(updateQuery, 
        user.Username, 
        user.Password, 
        user.Phone, 
        user.Address, 
        user.UserID)
    if err != nil {
        return fmt.Errorf("更新数据库失败: %v", err)
    }

    // 提交事务
    if err = tx.Commit(); err != nil {
        return fmt.Errorf("提交事务失败: %v", err)
    }

    // 更新Redis缓存
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() //创建一个超时控制上下文对象
    rdb := rp.GetClient()
    defer rp.PutClient(rdb)
    
    err = rdb.HMSet(ctx,
		fmt.Sprintf("user:%d", user.UserID),  //使用user_id作为Redis的key
		map[string]interface{}{    //哈希表内容，key-value对
        "user_id":  user.UserID,
        "username": user.Username,
        "password": user.Password,
        "phone":    user.Phone,
        "address":  user.Address,
    }).Err()
    if err != nil {
        // Redis更新失败不影响主流程
        log.Printf("警告: 更新Redis缓存失败: %v", err)
    }

    return nil
}


//获取所有用户(分页查询用户)，为了查看所有用户，监控用户状态，分析用户分布等等需要所有用户的数据时候用得到
func GetAllUsers(offset, limit int) ([]models.User, error) {

	var users []models.User  //因为要返回多个用户数据，所以设计成User类型的切片
	query := `SELECT userid, username, userphone, useraddress
			FROM users 
			LIMIT ? OFFSET ?`
			//OFFSET = (页码 - 1) * 每页数量   LIMIT = 每页数量
	
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User    // 创建单个用户变量
		err := rows.Scan(&user.UserID, &user.UserName, &user.UserPhone, &user.UserAddress) // 填充用户数据
		if err != nil {
			return nil, err
		}
		users = append(users, user)  // 将用户添加到切片中，append可以自动扩容
	}
	return users, nil
} 