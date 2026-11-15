package refund

import (
	"database/sql"
	"errors"

	"github.com/ovo133838-lang/scene/internal/order"
)

// ApplyRefund 买家对已付款未发货订单申请整单退款：PAID -> REFUND_PENDING，暂停发货。
func (s *Service) ApplyRefund(orderID string, buyerID int64, reason string) (*Refund, error) {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var obuyer, pid int64
	var qty int
	var total int64
	var status string
	err = tx.QueryRow("SELECT buyer_id, product_id, quantity, total_amount, status FROM `order` WHERE id = ? FOR UPDATE", orderID).
		Scan(&obuyer, &pid, &qty, &total, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, order.ErrUnknownOrder
	}
	if err != nil {
		return nil, err
	}
	if obuyer != buyerID {
		return nil, order.ErrNotOwner
	}
	if status != order.StatusPaid {
		return nil, ErrNotRefundable
	}

	// 防重复申请：同一订单已有待审核/退款中的单
	var n int
	if err := tx.QueryRow("SELECT COUNT(*) FROM refund WHERE order_id = ? AND status IN ('pending','processing')", orderID).Scan(&n); err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, ErrDuplicateRefund
	}

	// 关联支付流水（售前退款）
	var paymentID *int64
	var pidVal int64
	err = tx.QueryRow("SELECT id FROM payment WHERE order_id = ? AND status = 'success' ORDER BY id DESC LIMIT 1", orderID).Scan(&pidVal)
	if err == nil {
		paymentID = &pidVal
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err := s.orders.Transition(tx, orderID, order.StatusPaid, order.StatusRefundPending); err != nil {
		return nil, err
	}

	res, err := tx.Exec(
		"INSERT INTO refund (order_id, payment_id, amount, type, status, reason) VALUES (?,?,?,?,?,?)",
		orderID, paymentID, total, TypeSale, StatusPending, reason,
	)
	if err != nil {
		return nil, err
	}
	refundID, _ := res.LastInsertId()

	if _, err := tx.Exec("INSERT INTO order_event (order_id, event, operator_id) VALUES (?, 'refund_applied', ?)", orderID, buyerID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &Refund{ID: refundID, OrderID: orderID, PaymentID: paymentID, Amount: total, Type: TypeSale, Status: StatusPending, Reason: reason}, nil
}
