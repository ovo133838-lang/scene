package product

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parseID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

// ListOnSaleHandler GET /api/products
func (s *Service) ListOnSaleHandler(c *gin.Context) {
	ps, err := s.ListOnSale()
	if err != nil {
		c.JSON(500, gin.H{"code": 50000, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": ps})
}

// GetHandler GET /api/products/:id
func (s *Service) GetHandler(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	p, err := s.Get(id)
	if err != nil {
		c.JSON(404, gin.H{"code": 40400, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": p})
}

// ListAllHandler GET /api/admin/products
func (s *Service) ListAllHandler(c *gin.Context) {
	ps, err := s.ListAll()
	if err != nil {
		c.JSON(500, gin.H{"code": 50000, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": ps})
}

// ShelfHandler POST /api/admin/products/:id/shelf
func (s *Service) ShelfHandler(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	if err := s.SetShelf(id, req.Status); err != nil {
		c.JSON(422, gin.H{"code": 42200, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok"})
}

// UpdateHandler PUT /api/admin/products/:id
func (s *Service) UpdateHandler(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	var req struct {
		Price *int64 `json:"price"`
		Stock *int   `json:"stock"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	if err := s.Update(id, req.Price, req.Stock); err != nil {
		c.JSON(422, gin.H{"code": 42200, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok"})
}

// StockHandler GET /api/admin/products/stock
func (s *Service) StockHandler(c *gin.Context) {
	ps, err := s.StockBoard()
	if err != nil {
		c.JSON(500, gin.H{"code": 50000, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": ps})
}
