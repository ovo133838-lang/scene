package order

import (
	"database/sql"
	"fmt"
)

// 订单状态枚举（与 docs/02-数据字典.md 一致）。
const (
	StatusPendingPayment   = "PENDING_PAYMENT"   // 待付款
	StatusPaid             = "PAID"              // 待发货（已付款确认）
	StatusRefundPending    = "REFUND_PENDING"    // 退款申请中
	StatusRefundProcessing = "REFUND_PROCESSING" // 退款处理中
	StatusRefundFailed     = "REFUND_FAILED"     // 退款失败
	StatusRefunded         = "REFUNDED"          // 已退款
	StatusShipped          = "SHIPPED"           // 已发货
	StatusClosed           = "CLOSED"            // 已关闭（取消/超时）
	StatusPaymentAbnormal  = "PAYMENT_ABNORMAL"  // 支付异常（迟到支付）
)

// transitions 定义合法状态迁移。
var transitions = map[string][]string{
	StatusPendingPayment:   {StatusPaid, StatusClosed},
	StatusPaid:             {StatusRefundPending, StatusShipped},
	StatusRefundPending:    {StatusPaid, StatusRefundProcessing},
	StatusRefundProcessing: {StatusRefunded, StatusRefundFailed},
	StatusRefundFailed:     {StatusRefundProcessing},
	StatusClosed:           {StatusPaymentAbnormal},
	StatusPaymentAbnormal:  {StatusRefunded},
}

// canTransition 判断 from -> to 是否合法。
func canTransition(from, to string) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// transition 在事务内以守卫 UPDATE 原子完成状态迁移，非法迁移返回错误。
func (s *Service) transition(tx *sql.Tx, orderID, from, to string) error {
	if !canTransition(from, to) {
		return fmt.Errorf("非法状态迁移 %s -> %s", from, to)
	}
	res, err := tx.Exec("UPDATE `order` SET status = ?, updated_at = NOW() WHERE id = ? AND status = ?", to, orderID, from)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("订单状态已变化，无法 %s -> %s", from, to)
	}
	return nil
}
