// 题目01 电商订单交易系统 · 后端入口
package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ovo133838-lang/scene/internal/auth"
	"github.com/ovo133838-lang/scene/internal/clock"
	"github.com/ovo133838-lang/scene/internal/mockpay"
	"github.com/ovo133838-lang/scene/internal/order"
	"github.com/ovo133838-lang/scene/internal/payment"
	"github.com/ovo133838-lang/scene/internal/product"
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

	// 商品模块
	p := product.New(s)
	api := r.Group("/api", a.Middleware())
	api.GET("/products", p.ListOnSaleHandler)
	api.GET("/products/:id", p.GetHandler)

	admin := r.Group("/api/admin", a.Middleware(), a.RequireRole("operator"))
	admin.GET("/products", p.ListAllHandler)
	admin.GET("/products/stock", p.StockHandler)
	admin.POST("/products/:id/shelf", p.ShelfHandler)
	admin.PUT("/products/:id", p.UpdateHandler)

	// 订单模块
	pendingTTL, _ := time.ParseDuration(cfg.Clock.PendingTTL)
	o := order.New(s, clock.New(), pendingTTL)
	api.POST("/orders", o.CreateOrderHandler)
	api.GET("/orders", o.ListHandler)
	api.GET("/orders/:id", o.GetHandler)
	api.POST("/orders/:id/cancel", o.CancelHandler)

	// 支付模块（发起 + 真实异步通知）
	pay := payment.New(o, cfg)
	api.POST("/orders/:id/pay", pay.PayHandler)
	r.POST("/pay/notify", pay.NotifyHandler)

	// 可控回调网关（运营/客服，异常注入）
	mock := mockpay.New(o, pay)
	payAdmin := r.Group("/api/admin/pay", a.Middleware(), a.RequireRole("operator", "service"))
	payAdmin.POST("/mock-notify", mock.MockNotifyHandler)

	// 超时清扫（服务端时间判断，重复关闭不多释放）
	go func() {
		for {
			time.Sleep(time.Second)
			if n, err := o.SweepExpired(); err == nil && n > 0 {
				log.Printf("sweep expired: %d orders closed", n)
			}
		}
	}()

	log.Printf("server listening on %s", cfg.Server.Addr)
	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
