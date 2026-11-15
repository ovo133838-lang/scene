// Package store 是数据访问层：MySQL/Redis 连接管理与配置加载。
// 业务包一律通过本层访问数据库，不直接持有连接。
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pelletier/go-toml/v2"
	"github.com/redis/go-redis/v9"
)

// Config 全局配置结构（与 configs/config.toml 对应）。
type Config struct {
	Server struct {
		Addr string
	}
	MySQL struct {
		Host         string
		Port         int
		User         string
		Password     string
		DBName       string
		MaxOpenConns int
		MaxIdleConns int
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	Alipay struct {
		AppID           string
		Gateway         string
		NotifyURL       string
		PrivateKey      string
		AlipayPublicKey string
	}
	Clock struct {
		PendingTTL string
	}
	JWT struct {
		Secret string
	}
}

// Store 聚合数据库与缓存连接。
type Store struct {
	DB  *sql.DB
	RDB *redis.Client
	Cfg *Config
}

// LoadConfig 按顺序加载并合并配置（后者覆盖前者），跳过不存在的文件。
func LoadConfig(paths ...string) (*Config, error) {
	cfg := &Config{}
	cfg.Server.Addr = ":8080"
	cfg.MySQL.MaxOpenConns = 20
	cfg.MySQL.MaxIdleConns = 5
	cfg.Clock.PendingTTL = "10m"
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read config %s: %w", p, err)
		}
		if err := toml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", p, err)
		}
	}
	return cfg, nil
}

// New 建立 MySQL 与 Redis 连接并做连通性检查。
func New(cfg *Config) (*Store, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Store{DB: db, RDB: rdb, Cfg: cfg}, nil
}

// Close 释放连接。
func (s *Store) Close() {
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.RDB != nil {
		_ = s.RDB.Close()
	}
}
