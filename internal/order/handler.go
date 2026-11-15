package order

import (
	"github.com/gin-gonic/gin"

	"github.com/ovo133838-lang/scene/internal/auth"
)

// CreateOrderHandler POST /api/orders
func (s *Service) CreateOrderHandler(c *gin.Context) {
	var req struct {
		ProductID int64 `json:"product_id"`
		Quantity  int   `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	o, err := s.CreateOrder(auth.UserID(c), req.ProductID, req.Quantity)
	if err != nil {
		c.JSON(422, gin.H{"code": 42200, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": o})
}

// ListHandler GET /api/orders
func (s *Service) ListHandler(c *gin.Context) {
	os, err := s.ListByBuyer(auth.UserID(c))
	if err != nil {
		c.JSON(500, gin.H{"code": 50000, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": os})
}

// GetHandler GET /api/orders/:id
func (s *Service) GetHandler(c *gin.Context) {
	o, err := s.Get(c.Param("id"), auth.UserID(c))
	if err != nil {
		c.JSON(404, gin.H{"code": 40400, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": o})
}

// CancelHandler POST /api/orders/:id/cancel
func (s *Service) CancelHandler(c *gin.Context) {
	if err := s.Cancel(c.Param("id"), auth.UserID(c)); err != nil {
		c.JSON(422, gin.H{"code": 42200, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok"})
}
