package database

import (
    "database/sql"
    "log"
    "time"
)

// 数据库监控统计结构
type DBStats struct {
    OpenConnections   int
    InUse              int
    Idle               int
    WaitCount          int64
    WaitDuration       time.Duration
    MaxIdleClosed      int64
    MaxLifetimeClosed  int64
}

// 监控连接池状态
func logDBStats(db *sql.DB) {
    if db != nil {
        stats := db.Stats()
        log.Printf("=== 数据库连接池状态 ===")
        log.Printf("打开连接数: %d", stats.OpenConnections)
        log.Printf("使用中连接数: %d", stats.InUse)
        log.Printf("空闲连接数: %d", stats.Idle)
        log.Printf("等待连接数: %d", stats.WaitCount)
        log.Printf("等待时长: %v", stats.WaitDuration)
        log.Printf("超过最大空闲时间关闭数: %d", stats.MaxIdleClosed)
        log.Printf("超过最大存活时间关闭数: %d", stats.MaxLifetimeClosed)
    }
}

// 启动定期监控
func StartDBMonitor(db *sql.DB, interval time.Duration) {
    ticker := time.NewTicker(interval)  //创建一个定时器，用于定期执行监控任务
    go func() {
        for range ticker.C {    // 每当到达间隔时间，就会执行循环体内的代码
            // 记录基本连接池状态
            logDBStats(db)
            
            // 检查连接健康状态
            if err := db.Ping(); err != nil {
                log.Printf("数据库连接异常: %v", err)
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
    utilization := float64(stats.InUse) / float64(stats.OpenConnections)
    if utilization > 0.8 { // 使用率超过 80%
        log.Printf("警告：连接池使用率较高 (%.2f%%)", utilization*100)
    }
}
