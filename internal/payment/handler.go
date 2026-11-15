package payment

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ovo133838-lang/scene/internal/auth"
	"github.com/ovo133838-lang/scene/internal/order"
)

// PayHandler POST /api/orders/:id/pay 发起支付
func (s *Service) PayHandler(c *gin.Context) {
	res, err := s.CreatePay(c.Param("id"), auth.UserID(c))
	if err != nil {
		status, code := orderErrStatus(err)
		c.JSON(status, gin.H{"code": code, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": res})
}

// NotifyHandler POST /pay/notify 支付宝沙箱异步通知（真实验签，幂等）。
// 响应约定：处理成功（含幂等/迟到登记）返回 "success"，否则 "fail"。
func (s *Service) NotifyHandler(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(400, "fail")
		return
	}
	params := make(map[string]string, len(c.Request.PostForm))
	for k, v := range c.Request.PostForm {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 1. 验签：未配置公钥或验签失败一律拒绝
	if s.cfg.Alipay.AlipayPublicKey == "" {
		c.String(200, "fail")
		return
	}
	if err := verifyRSA2(s.cfg.Alipay.AlipayPublicKey, signSource(params), params["sign"]); err != nil {
		c.String(200, "fail")
		return
	}

	// 2. 仅处理成功类通知，其余状态无需处理
	if status := params["trade_status"]; status != "TRADE_SUCCESS" && status != "TRADE_FINISHED" {
		c.String(200, "success")
		return
	}

	// 3. 解析订单号、渠道交易号、金额
	orderID := params["out_trade_no"]
	txnNo := params["trade_no"]
	amount, err := yuanToFen(params["total_amount"])
	if err != nil {
		c.String(200, "fail")
		return
	}

	// 4. 处理（幂等/迟到支付在此收敛）
	res := s.ProcessNotify(orderID, txnNo, amount)
	if res.Code == "ok" || res.Code == "late_payment" {
		c.String(200, "success")
		return
	}
	c.String(200, "fail")
}

// orderErrStatus 将 order 域错误映射为 HTTP 状态与业务码。
func orderErrStatus(err error) (int, int) {
	switch {
	case errors.Is(err, order.ErrUnknownOrder):
		return 404, 40400
	case errors.Is(err, order.ErrNotOwner):
		return 403, 40300
	case errors.Is(err, order.ErrNotPayable):
		return 409, 40900
	default:
		return 500, 50000
	}
}
