package client

import (
	"database/sql"
	"log/slog"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbPath = "cdn_cache.db"
	db     *sql.DB
	dbOnce sync.Once
)

// SetDBPath 设置数据库路径
func SetDBPath(path string) {
	if path != "" {
		if !strings.HasSuffix(path, ".db") {
			path += "cdn_cache.db"
		}
		dbPath = path
	}
}

// initDB 初始化数据库连接，确保只执行一次
func initDB() (*sql.DB, error) {
	var err error
	dbOnce.Do(func() {
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			slog.Error("初始化数据库失败:", err)
			return
		}

		// 创建表
		createTableSQL := `
		CREATE TABLE IF NOT EXISTS cdn_cache (
			ip TEXT PRIMARY KEY,
			location TEXT,
			source TEXT,
			report_time DATETIME
		);
		`
		_, err = db.Exec(createTableSQL)
		if err != nil {
			slog.Error("创建表失败:", err)
			return
		}
	})
	return db, err
}

// CheckIPInCache 检查IP是否在缓存中，并且数据是否在3个月内
func CheckIPInCache(ip string) (bool, string, string, error) {
	db, err := initDB()
	if err != nil {
		return false, "", "", err
	}

	var location, source string
	var reportTime time.Time
	err = db.QueryRow("SELECT location, source, report_time FROM cdn_cache WHERE ip = ?", ip).Scan(&location, &source, &reportTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", "", nil
		}
		return false, "", "", err
	}

	// 检查数据是否在3个月内
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	if reportTime.Before(threeMonthsAgo) {
		return false, "", "", nil
	}

	return true, location, source, nil
}

// InsertIPToCache 将IP信息插入缓存
func InsertIPToCache(ip, location, source string) {
	db, err := initDB()
	if err != nil {
		slog.Error("初始化db失败:", err)
		return
	}

	reportTime := time.Now()
	_, err = db.Exec("INSERT OR REPLACE INTO cdn_cache (ip, location, source, report_time) VALUES (?, ?, ?, ?)", ip, location, source, reportTime)
	if err != nil {
		slog.Error("数据db插入异常:", err)
	}
}
