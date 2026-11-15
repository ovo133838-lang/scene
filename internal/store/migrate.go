package store

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// Migrate 执行数据库迁移：建库、建表、种子数据（幂等，可重复执行）。
func Migrate(cfg *Config) error {
	// 1. 先不带库名连接，执行 schema.sql（含 CREATE DATABASE + USE + 建表）
	base := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port)
	db0, err := sql.Open("mysql", base)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}
	defer db0.Close()
	if err := execSQLFile(db0, "migrations/schema.sql"); err != nil {
		return fmt.Errorf("schema: %w", err)
	}

	// 2. 再带库名连接，执行 seed.sql
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := execSQLFile(db, "migrations/seed.sql"); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	return nil
}

// execSQLFile 读取 SQL 文件，按分号拆分并逐条执行（跳过空语句与 -- 注释）。
func execSQLFile(db *sql.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for i, stmt := range splitStatements(string(data)) {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("stmt %d: %w", i+1, err)
		}
	}
	return nil
}

func splitStatements(text string) []string {
	var out []string
	for _, raw := range strings.Split(text, ";") {
		var lines []string
		for _, l := range strings.Split(raw, "\n") {
			t := strings.TrimSpace(l)
			if t == "" || strings.HasPrefix(t, "--") {
				continue
			}
			lines = append(lines, l)
		}
		s := strings.TrimSpace(strings.Join(lines, "\n"))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
