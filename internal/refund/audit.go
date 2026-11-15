package refund

import (
	"database/sql"
	"errors"

	"github.com/ovo133838-lang/scene/internal/order"
)

// refundRow 锁定后的退款单关键字段。
type refundRow struct {
	id         int64
	orderID    string
	amount     int64
	typ        string
	status     string
	retryCount int
}

// ApproveRefund 客服同意退款：REFUND_PENDING -> REFUND_PROCESSING ->（渠道结果）REFUNDED/REFUND_FAILED。
func (s *Service) ApproveRefund(refundID int64, remark string) error {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r, err := s.lockRefund(tx, refundID)
	if err != nil {
		return err
	}
	if r.status != StatusPending {
		return ErrRefundState
	}
	pid, qty, ostatus, err := s.lockOrderForRefund(tx, r.orderID)
	if err != nil {
		return err
	}
	if ostatus != order.StatusRefundPending {
		return ErrRefundState
	}

	if err := s.orders.Transition(tx, r.orderID, order.StatusRefundPending, order.StatusRefundProcessing); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE refund SET status = ?, updated_at = NOW() WHERE id = ?", StatusProcessing, refundID); err != nil {
		return err
	}
	if err := s.settle(tx, refundID, r.orderID, pid, qty, r.amount, r.retryCount, remark); err != nil {
		return err
	}
	return tx.Commit()
}

// RejectRefund 客服拒绝退款：REFUND_PENDING -> PAID（恢复可发货）。
func (s *Service) RejectRefund(refundID int64, reason string) error {
	if reason == "" {
		return ErrReasonRequired
	}
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r, err := s.lockRefund(tx, refundID)
	if err != nil {
		return err
	}
	if r.status != StatusPending {
		return ErrRefundState
	}
	_, _, ostatus, err := s.lockOrderForRefund(tx, r.orderID)
	if err != nil {
		return err
	}
	if ostatus != order.StatusRefundPending {
		return ErrRefundState
	}

	if err := s.orders.Transition(tx, r.orderID, order.StatusRefundPending, order.StatusPaid); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE refund SET status = ?, reason = ?, updated_at = NOW() WHERE id = ?", StatusRejected, reason, refundID); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO order_event (order_id, event, detail) VALUES (?, 'refund_rejected', ?)", r.orderID, reason); err != nil {
		return err
	}
	return tx.Commit()
}

// RetryRefund 退款重试：REFUND_FAILED -> REFUND_PROCESSING ->（渠道结果）REFUNDED/REFUND_FAILED。
func (s *Service) RetryRefund(refundID int64) error {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r, err := s.lockRefund(tx, refundID)
	if err != nil {
		return err
	}
	if r.status != StatusFailed {
		return ErrRefundState
	}
	pid, qty, ostatus, err := s.lockOrderForRefund(tx, r.orderID)
	if err != nil {
		return err
	}
	if ostatus != order.StatusRefundFailed {
		return ErrRefundState
	}

	if err := s.orders.Transition(tx, r.orderID, order.StatusRefundFailed, order.StatusRefundProcessing); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE refund SET status = ?, updated_at = NOW() WHERE id = ?", StatusProcessing, refundID); err != nil {
		return err
	}
	if err := s.settle(tx, refundID, r.orderID, pid, qty, r.amount, r.retryCount, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// settle 依据渠道退款结果推进订单与退款单状态；售前退款成功恢复库存。
func (s *Service) settle(tx *sql.Tx, refundID int64, orderID string, pid int64, qty int, amount int64, retryCount int, remark string) error {
	if err := s.doChannelRefund(orderID, amount, retryCount); err == nil {
		if e := s.orders.Transition(tx, orderID, order.StatusRefundProcessing, order.StatusRefunded); e != nil {
			return e
		}
		if _, e := tx.Exec("UPDATE product SET stock = stock + ? WHERE id = ?", qty, pid); e != nil {
			return e
		}
		if _, e := tx.Exec("UPDATE refund SET status = ?, updated_at = NOW() WHERE id = ?", StatusSuccess, refundID); e != nil {
			return e
		}
		if _, e := tx.Exec("INSERT INTO order_event (order_id, event, detail) VALUES (?, 'refund_success', ?)", orderID, remark); e != nil {
			return e
		}
		return nil
	}

	// 渠道明确失败：保留待处理记录，可重试
	if e := s.orders.Transition(tx, orderID, order.StatusRefundProcessing, order.StatusRefundFailed); e != nil {
		return e
	}
	if _, e := tx.Exec("UPDATE refund SET status = ?, retry_count = retry_count + 1, updated_at = NOW() WHERE id = ?", StatusFailed, refundID); e != nil {
		return e
	}
	if _, e := tx.Exec("INSERT INTO order_event (order_id, event) VALUES (?, 'refund_failed')", orderID); e != nil {
		return e
	}
	return nil
}

// lockRefund 锁定退款单行。
func (s *Service) lockRefund(tx *sql.Tx, refundID int64) (*refundRow, error) {
	var r refundRow
	err := tx.QueryRow("SELECT id, order_id, amount, type, status, retry_count FROM refund WHERE id = ? FOR UPDATE", refundID).
		Scan(&r.id, &r.orderID, &r.amount, &r.typ, &r.status, &r.retryCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRefundNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// lockOrderForRefund 锁定订单行，返回商品与数量用于恢复库存。
func (s *Service) lockOrderForRefund(tx *sql.Tx, orderID string) (pid int64, qty int, status string, err error) {
	err = tx.QueryRow("SELECT product_id, quantity, status FROM `order` WHERE id = ? FOR UPDATE", orderID).Scan(&pid, &qty, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, "", order.ErrUnknownOrder
	}
	if err != nil {
		return 0, 0, "", err
	}
	return pid, qty, status, nil
}
