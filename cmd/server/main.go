// 题目01 电商订单交易系统 · 后端入口
package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ovo133838-lang/scene/internal/auth"
	"github.com/ovo133838-lang/scene/internal/store"
)

func main() {
	cfg, err := store.LoadConfig("configs/config.toml", "configs/config.local.toml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := store.Migrate(cfg); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	s, err := store.New(cfg)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer s.Close()

	a := auth.New(s, cfg.JWT.Secret)
	if err := a.SeedUsers(); err != nil {
		log.Fatalf("seed users: %v", err)
	}

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.POST("/api/login", a.LoginHandler)

	log.Printf("server listening on %s", cfg.Server.Addr)
	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
