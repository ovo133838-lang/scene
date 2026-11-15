package refund

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ovo133838-lang/scene/internal/auth"
	"github.com/ovo133838-lang/scene/internal/order"
)

// ApplyRefundHandler POST /api/orders/:id/refund 申请售前退款
func (s *Service) ApplyRefundHandler(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	r, err := s.ApplyRefund(c.Param("id"), auth.UserID(c), req.Reason)
	if err != nil {
		status, code := refundErrStatus(err)
		c.JSON(status, gin.H{"code": code, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": r})
}

// ListPendingRefundsHandler GET /api/admin/refunds/pending 退款待审核
func (s *Service) ListPendingRefundsHandler(c *gin.Context) {
	list, err := s.ListPendingRefunds()
	if err != nil {
		c.JSON(500, gin.H{"code": 50000, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": list})
}

// ApproveRefundHandler POST /api/admin/refunds/:id/approve 同意退款
func (s *Service) ApproveRefundHandler(c *gin.Context) {
	id, err := parseRefundID(c)
	if err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	var req struct {
		Remark string `json:"remark"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := s.ApproveRefund(id, req.Remark); err != nil {
		status, code := refundErrStatus(err)
		c.JSON(status, gin.H{"code": code, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok"})
}

// RejectRefundHandler POST /api/admin/refunds/:id/reject 拒绝退款
func (s *Service) RejectRefundHandler(c *gin.Context) {
	id, err := parseRefundID(c)
	if err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	if err := s.RejectRefund(id, req.Reason); err != nil {
		status, code := refundErrStatus(err)
		c.JSON(status, gin.H{"code": code, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok"})
}

// RetryRefundHandler POST /api/admin/refunds/:id/retry 退款重试
func (s *Service) RetryRefundHandler(c *gin.Context) {
	id, err := parseRefundID(c)
	if err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	if err := s.RetryRefund(id); err != nil {
		status, code := refundErrStatus(err)
		c.JSON(status, gin.H{"code": code, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok"})
}

// ListAbnormalHandler GET /api/admin/pay-abnormal 异常支付列表
func (s *Service) ListAbnormalHandler(c *gin.Context) {
	list, err := s.ListAbnormal()
	if err != nil {
		c.JSON(500, gin.H{"code": 50000, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": list})
}

// RefundLatePaymentHandler POST /api/admin/pay-abnormal/:id/refund 异常支付退款（不恢复库存）
func (s *Service) RefundLatePaymentHandler(c *gin.Context) {
	id, err := parseRefundID(c)
	if err != nil {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	if err := s.RefundLatePayment(id); err != nil {
		status, code := refundErrStatus(err)
		c.JSON(status, gin.H{"code": code, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok"})
}

func parseRefundID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

func refundErrStatus(err error) (int, int) {
	switch {
	case errors.Is(err, order.ErrUnknownOrder), errors.Is(err, ErrRefundNotFound), errors.Is(err, ErrAbnormalNotFound):
		return 404, 40400
	case errors.Is(err, order.ErrNotOwner):
		return 403, 40300
	case errors.Is(err, ErrReasonRequired):
		return 400, 40000
	case errors.Is(err, ErrNotRefundable), errors.Is(err, ErrDuplicateRefund),
		errors.Is(err, ErrRefundState), errors.Is(err, ErrAbnormalState),
		errors.Is(err, order.ErrInvalidState):
		return 409, 40900
	default:
		return 500, 50000
	}
}
