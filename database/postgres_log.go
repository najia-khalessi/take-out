package database

import (
	"database/sql"
	"take-out/logging"
	"time"

	"github.com/sirupsen/logrus"
)

// 数据库监控统计结构
type DBStats struct {
	OpenConnections   int
	InUse             int
	Idle              int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
}

// 监控连接池状态
func logDBStats(db *sql.DB) {
	if db != nil {
		stats := db.Stats()
		logging.Info("数据库连接池状态", logrus.Fields{
			"open_connections":    stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"wait_count":          stats.WaitCount,
			"wait_duration":       stats.WaitDuration,
			"max_idle_closed":     stats.MaxIdleClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		})
	}
}

// 启动定期监控
func StartDBMonitor(db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval) //创建一个定时器，用于定期执行监控任务
	go func() {
		for range ticker.C { // 每当到达间隔时间，就会执行循环体内的代码
			// 记录基本连接池状态
			logDBStats(db)

			// 检查连接健康状态
			if err := db.Ping(); err != nil {
				logging.Error("数据库连接异常", logrus.Fields{"error": err})
				// 可以在这里添加告警逻辑
			}
			// 检查连接池使用率
			checkPoolUtilization(db)
		}
	}()
}

// 检查连接池使用率
func checkPoolUtilization(db *sql.DB) {
	stats := db.Stats()
	if stats.OpenConnections > 0 {
		utilization := float64(stats.InUse) / float64(stats.OpenConnections)
		if utilization > 0.8 { // 使用率超过 80%
			logging.Warn("连接池使用率较高", logrus.Fields{"utilization": utilization})
		}
	}
}