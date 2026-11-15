// Package mockpay 可控回调网关：注入成功/失败/超时/重复/迟到/验签失败/金额不符/未知订单等通知，
// 复用 payment 域同一套业务规则（ProcessNotify），仅教学演示与异常场景测试使用。
package mockpay

import (
	"github.com/gin-gonic/gin"

	"github.com/ovo133838-lang/scene/internal/order"
	"github.com/ovo133838-lang/scene/internal/payment"
)

// Service 可控回调网关。
type Service struct {
	orders *order.Service
	pay    *payment.Service
}

// New 创建回调网关。
func New(o *order.Service, p *payment.Service) *Service {
	return &Service{orders: o, pay: p}
}

// MockNotifyHandler POST /api/admin/pay/mock-notify
// 请求体：{ "out_trade_no":"...", "amount":10000, "result":"success|fail", "mode":"repeat|late|bad_sign|wrong_amount|unknown" }
func (s *Service) MockNotifyHandler(c *gin.Context) {
	var req struct {
		OutTradeNo string `json:"out_trade_no"`
		Amount     int64  `json:"amount"`
		Result     string `json:"result"` // success / fail
		Mode       string `json:"mode"`   // repeat / late / bad_sign / wrong_amount / unknown
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OutTradeNo == "" {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	if req.Result == "" {
		req.Result = "success"
	}

	txnNo := "MOCK" + req.OutTradeNo

	switch req.Mode {
	case "bad_sign":
		// 验签失败：直接拒绝，不进入业务处理
		c.JSON(200, gin.H{"code": 42200, "message": "验签失败", "data": gin.H{"mode": "bad_sign"}})
		return
	case "unknown":
		// 未知订单：order_id 不存在
		res := s.pay.ProcessNotify(req.OutTradeNo, txnNo, req.Amount)
		c.JSON(200, gin.H{"code": 42200, "message": "未知订单", "data": res})
		return
	case "wrong_amount":
		// 金额不符：以错误金额注入
		res := s.pay.ProcessNotify(req.OutTradeNo, txnNo, req.Amount+1)
		c.JSON(200, gin.H{"code": 42200, "message": "金额不符", "data": res})
		return
	case "late":
		// 迟到支付：先保证订单已关闭（模拟超时），再注入成功通知
		if err := s.orders.Expire(req.OutTradeNo); err != nil {
			c.JSON(422, gin.H{"code": 42200, "message": err.Error()})
			return
		}
		if req.Result == "fail" {
			c.JSON(200, gin.H{"code": 0, "message": "迟到且失败通知：订单保持关闭", "data": gin.H{"mode": "late", "result": "fail"}})
			return
		}
		res := s.pay.ProcessNotify(req.OutTradeNo, txnNo, req.Amount)
		c.JSON(200, gin.H{"code": 0, "message": "迟到支付", "data": res})
		return
	}

	// 默认：成功 / 失败 / 重复通知
	if req.Result == "fail" {
		c.JSON(200, gin.H{"code": 0, "message": "支付失败通知：订单保持待付款", "data": gin.H{"mode": "fail"}})
		return
	}

	if req.Mode == "repeat" {
		// 重复注入两次，验证幂等（第二次不重复记款）
		first := s.pay.ProcessNotify(req.OutTradeNo, txnNo, req.Amount)
		second := s.pay.ProcessNotify(req.OutTradeNo, txnNo, req.Amount)
		c.JSON(200, gin.H{"code": 0, "message": "重复通知（幂等）", "data": gin.H{"first": first, "second": second}})
		return
	}

	res := s.pay.ProcessNotify(req.OutTradeNo, txnNo, req.Amount)
	c.JSON(200, gin.H{"code": 0, "message": "成功通知", "data": res})
}
