package order

import (
	"database/sql"
	"errors"
)

// 支付通知处理的哨兵结果。
var (
	ErrUnknownOrder   = errors.New("未知订单")
	ErrAmountMismatch = errors.New("金额不符")
	ErrTxnConflict    = errors.New("交易号归属冲突")
	ErrOrderClosed    = errors.New("订单已关闭")
	ErrInvalidState   = errors.New("非法订单状态")
	ErrNotOwner       = errors.New("无权访问该订单")
	ErrNotPayable     = errors.New("当前状态不可支付")
)

// MarkPaid 支付成功确认：PENDING_PAYMENT -> PAID，落支付流水。
// 幂等与冲突由 channel_txn_no 唯一约束保证；重复通知返回 nil（幂等）。
func (s *Service) MarkPaid(orderID, channelTxnNo string, amount int64) error {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var total int64
	var status string
	err = tx.QueryRow("SELECT total_amount, status FROM `order` WHERE id = ? FOR UPDATE", orderID).Scan(&total, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnknownOrder
	}
	if err != nil {
		return err
	}
	if total != amount {
		return ErrAmountMismatch
	}

	var existingOrder string
	err = tx.QueryRow("SELECT order_id FROM payment WHERE channel_txn_no = ? FOR UPDATE", channelTxnNo).Scan(&existingOrder)
	if err == nil {
		if existingOrder == orderID {
			return nil // 重复通知，幂等
		}
		return ErrTxnConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	switch status {
	case StatusPendingPayment:
		if err := s.transition(tx, orderID, StatusPendingPayment, StatusPaid); err != nil {
			return err
		}
	case StatusPaid, StatusPaymentAbnormal:
		return nil // 已确认/已登记异常，兜底幂等
	case StatusClosed:
		return ErrOrderClosed // 迟到支付，由支付域转入异常登记
	default:
		return ErrInvalidState
	}

	if _, err := tx.Exec("INSERT INTO payment (order_id, channel_txn_no, amount, status, notify_at) VALUES (?,?,?,'success',NOW())", orderID, channelTxnNo, amount); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO order_event (order_id, event) VALUES (?, 'paid')", orderID); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkPaymentAbnormal 迟到支付登记：CLOSED -> PAYMENT_ABNORMAL，不恢复订单/库存。
func (s *Service) MarkPaymentAbnormal(orderID, channelTxnNo string, amount int64) error {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRow("SELECT status FROM `order` WHERE id = ? FOR UPDATE", orderID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnknownOrder
	}
	if err != nil {
		return err
	}
	switch status {
	case StatusClosed:
		// 迟到支付：关闭 -> 支付异常
	case StatusPaymentAbnormal:
		return nil // 已登记过异常，幂等
	default:
		return ErrInvalidState
	}
	if err := s.transition(tx, orderID, StatusClosed, StatusPaymentAbnormal); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO payment_abnormal (order_id, channel_txn_no, amount, status, reason) VALUES (?,?,?,'pending','late_payment')", orderID, channelTxnNo, amount); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO order_event (order_id, event) VALUES (?, 'payment_abnormal')", orderID); err != nil {
		return err
	}
	return tx.Commit()
}

// PreparePay 发起支付前置校验：仅本人待付款订单可发起支付。
// 区分「订单不存在」「越权」「状态不可支付」三类结果，便于接入层映射 404/403/409。
func (s *Service) PreparePay(orderID string, buyerID int64) (*Order, error) {
	row := s.store.DB.QueryRow("SELECT "+orderCols+" FROM `order` WHERE id = ?", orderID)
	o, err := scanOrder(row)
	if err != nil {
		return nil, ErrUnknownOrder
	}
	if o.BuyerID != buyerID {
		return nil, ErrNotOwner
	}
	if o.Status != StatusPendingPayment {
		return nil, ErrNotPayable
	}
	return o, nil
}
