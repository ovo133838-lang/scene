package payment

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ovo133838-lang/scene/internal/order"
)

// NotifyResult 通知处理结果，用于响应与异常注入后的可查询反馈。
type NotifyResult struct {
	Code    string `json:"code"` // ok / late_payment / unknown_order / amount_mismatch / txn_conflict / invalid_state / error
	Message string `json:"message"`
}

// ProcessNotify 处理已验签通过的支付成功通知：匹配订单+金额+幂等+状态迁移/迟到支付。
// 该方法是真实回调与可控回调网关共用的唯一业务入口，保证测试与演示同规则。
func (s *Service) ProcessNotify(orderID, channelTxnNo string, amount int64) NotifyResult {
	err := s.orders.MarkPaid(orderID, channelTxnNo, amount)
	if err == nil {
		return NotifyResult{Code: "ok", Message: "支付成功确认"}
	}
	if errors.Is(err, order.ErrOrderClosed) {
		// 迟到支付：关闭订单 -> 支付异常待处理，不恢复订单/库存
		if e2 := s.orders.MarkPaymentAbnormal(orderID, channelTxnNo, amount); e2 != nil {
			return mapNotifyError(e2)
		}
		return NotifyResult{Code: "late_payment", Message: "迟到支付已登记异常待处理"}
	}
	return mapNotifyError(err)
}

// mapNotifyError 将 order 域哨兵错误映射为可查询结果。
func mapNotifyError(err error) NotifyResult {
	switch {
	case errors.Is(err, order.ErrUnknownOrder):
		return NotifyResult{Code: "unknown_order", Message: "未知订单"}
	case errors.Is(err, order.ErrAmountMismatch):
		return NotifyResult{Code: "amount_mismatch", Message: "金额不符"}
	case errors.Is(err, order.ErrTxnConflict):
		return NotifyResult{Code: "txn_conflict", Message: "交易号归属冲突"}
	case errors.Is(err, order.ErrInvalidState):
		return NotifyResult{Code: "invalid_state", Message: "非法订单状态"}
	default:
		return NotifyResult{Code: "error", Message: err.Error()}
	}
}

// yuanToFen 元字符串转分（两位小数），解析失败返回错误。
func yuanToFen(s string) (int64, error) {
	parts := strings.SplitN(s, ".", 2)
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	var fen int64
	if len(parts) == 2 && parts[1] != "" {
		frac := parts[1]
		for len(frac) < 2 {
			frac += "0"
		}
		frac = frac[:2]
		f, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, err
		}
		fen = f
	}
	return yuan*100 + fen, nil
}
