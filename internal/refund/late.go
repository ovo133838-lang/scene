package refund

import (
	"database/sql"
	"errors"

	"github.com/ovo133838-lang/scene/internal/order"
)

// ListPendingRefunds 返回待审核退款列表（客服）。
func (s *Service) ListPendingRefunds() ([]Refund, error) {
	rows, err := s.store.DB.Query("SELECT id, order_id, payment_id, amount, type, status, reason, retry_count, created_at, updated_at FROM refund WHERE status = ? ORDER BY created_at", StatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefunds(rows)
}

// ListAbnormal 返回异常支付待处理列表（客服）。
func (s *Service) ListAbnormal() ([]Abnormal, error) {
	rows, err := s.store.DB.Query("SELECT id, order_id, channel_txn_no, amount, status, reason, created_at FROM payment_abnormal WHERE status = 'pending' ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Abnormal
	for rows.Next() {
		var a Abnormal
		if err := rows.Scan(&a.ID, &a.OrderID, &a.ChannelTxnNo, &a.Amount, &a.Status, &a.Reason, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// RefundLatePayment 迟到支付退款：PAYMENT_ABNORMAL -> REFUNDED，不恢复库存（已释放过）。
func (s *Service) RefundLatePayment(abnormalID int64) error {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var orderID string
	var amount int64
	var status string
	err = tx.QueryRow("SELECT order_id, amount, status FROM payment_abnormal WHERE id = ? FOR UPDATE", abnormalID).Scan(&orderID, &amount, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAbnormalNotFound
	}
	if err != nil {
		return err
	}
	if status != "pending" {
		return ErrAbnormalState
	}

	var ostatus string
	if err := tx.QueryRow("SELECT status FROM `order` WHERE id = ? FOR UPDATE", orderID).Scan(&ostatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return order.ErrUnknownOrder
		}
		return err
	}
	if ostatus != order.StatusPaymentAbnormal {
		return ErrAbnormalState
	}

	if err := s.orders.Transition(tx, orderID, order.StatusPaymentAbnormal, order.StatusRefunded); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE payment_abnormal SET status = 'refunded' WHERE id = ?", abnormalID); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO refund (order_id, amount, type, status, reason) VALUES (?,?,?,?,?)", orderID, amount, TypeLatePayment, StatusSuccess, "迟到支付退款"); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO order_event (order_id, event) VALUES (?, 'late_refunded')", orderID); err != nil {
		return err
	}
	return tx.Commit()
}

func scanRefunds(rows *sql.Rows) ([]Refund, error) {
	var out []Refund
	for rows.Next() {
		var r Refund
		var updatedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.OrderID, &r.PaymentID, &r.Amount, &r.Type, &r.Status, &r.Reason, &r.RetryCount, &r.CreatedAt, &updatedAt); err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			r.UpdatedAt = updatedAt.Time
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
