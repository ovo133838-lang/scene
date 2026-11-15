// Package order 订单域：下单原子占用库存、状态机、取消与超时释放。
package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ovo133838-lang/scene/internal/clock"
	"github.com/ovo133838-lang/scene/internal/store"
)

// Order 订单。
type Order struct {
	ID            string    `json:"id"`
	BuyerID       int64     `json:"buyer_id"`
	ProductID     int64     `json:"product_id"`
	ProductName   string    `json:"product_name"`
	PriceSnapshot int64     `json:"price_snapshot"`
	Quantity      int       `json:"quantity"`
	TotalAmount   int64     `json:"total_amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiredAt     time.Time `json:"expired_at"`
}

// Service 订单域服务。
type Service struct {
	store      *store.Store
	clock      *clock.Clock
	pendingTTL time.Duration
}

// New 创建订单服务。
func New(s *store.Store, c *clock.Clock, pendingTTL time.Duration) *Service {
	return &Service{store: s, clock: c, pendingTTL: pendingTTL}
}

const orderCols = `id, buyer_id, product_id, product_name, price_snapshot, quantity, total_amount, status, created_at, expired_at`

// CreateOrder 下单：事务内锁商品行 + 条件扣库存（防超卖），保存价格快照。
func (s *Service) CreateOrder(buyerID, productID int64, quantity int) (*Order, error) {
	if quantity < 1 || quantity > 2 {
		return nil, errors.New("数量须为 1~2 件")
	}
	tx, err := s.store.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var name string
	var price int64
	var status string
	err = tx.QueryRow(`SELECT name, price, status FROM product WHERE id = ? FOR UPDATE`, productID).Scan(&name, &price, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("商品不存在")
	}
	if err != nil {
		return nil, err
	}
	if status != "on" {
		return nil, errors.New("商品已下架")
	}

	res, err := tx.Exec(`UPDATE product SET stock = stock - ? WHERE id = ? AND stock >= ?`, quantity, productID, quantity)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, errors.New("库存不足")
	}

	orderID, err := s.genOrderID()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	total := price * int64(quantity)
	_, err = tx.Exec(
		"INSERT INTO `order` ("+orderCols+") VALUES (?,?,?,?,?,?,?,?,?,?)",
		orderID, buyerID, productID, name, price, quantity, total, StatusPendingPayment, now, now.Add(s.pendingTTL),
	)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`INSERT INTO order_event (order_id, event, operator_id) VALUES (?, 'created', ?)`, orderID, buyerID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(orderID, buyerID)
}

// Cancel 取消本人待付款订单并释放库存（守卫保证只释放一次）。
func (s *Service) Cancel(orderID string, buyerID int64) error {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var pid int64
	var qty int
	var status string
	err = tx.QueryRow(`SELECT product_id, quantity, status FROM `+"`order`"+` WHERE id = ? AND buyer_id = ? FOR UPDATE`, orderID, buyerID).Scan(&pid, &qty, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("订单不存在")
	}
	if err != nil {
		return err
	}
	if status != StatusPendingPayment {
		return errors.New("当前状态不可取消")
	}

	if err := s.transition(tx, orderID, StatusPendingPayment, StatusClosed); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE product SET stock = stock + ? WHERE id = ?`, qty, pid); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO order_event (order_id, event, operator_id) VALUES (?, 'cancelled', ?)`, orderID, buyerID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListByBuyer 返回买家订单列表。
func (s *Service) ListByBuyer(buyerID int64) ([]Order, error) {
	rows, err := s.store.DB.Query("SELECT "+orderCols+" FROM `order` WHERE buyer_id = ? ORDER BY created_at DESC, id DESC", buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

// Get 返回订单详情（校验归属）。
func (s *Service) Get(orderID string, buyerID int64) (*Order, error) {
	row := s.store.DB.QueryRow("SELECT "+orderCols+" FROM `order` WHERE id = ? AND buyer_id = ?", orderID, buyerID)
	return scanOrder(row)
}

// SweepExpired 超时关闭待付款订单并释放库存，返回关闭数量。
func (s *Service) SweepExpired() (int, error) {
	now := s.clock.Now()
	rows, err := s.store.DB.Query("SELECT id, product_id, quantity FROM `order` WHERE status = ? AND expired_at <= ?", StatusPendingPayment, now)
	if err != nil {
		return 0, err
	}
	type item struct {
		id  string
		pid int64
		qty int
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.pid, &it.qty); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()

	n := 0
	for _, it := range items {
		if s.closeExpired(it) == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) closeExpired(it struct {
	id  string
	pid int64
	qty int
}) error {
	tx, err := s.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// 守卫：仅当仍为待付款时才关闭，避免重复释放
	res, err := tx.Exec("UPDATE `order` SET status = ?, updated_at = NOW() WHERE id = ? AND status = ?", StatusClosed, it.id, StatusPendingPayment)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil // 已被处理，跳过
	}
	if _, err := tx.Exec(`UPDATE product SET stock = stock + ? WHERE id = ?`, it.qty, it.pid); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO order_event (order_id, event) VALUES (?, 'timeout_closed')`, it.id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) genOrderID() (string, error) {
	date := s.clock.Now().Format("20060102")
	seq, err := s.store.RDB.Incr(context.Background(), "order_seq:"+date).Result()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("T01%s%06d", date, seq), nil
}

func scanOrder(row interface{ Scan(...interface{}) error }) (*Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.BuyerID, &o.ProductID, &o.ProductName, &o.PriceSnapshot, &o.Quantity, &o.TotalAmount, &o.Status, &o.CreatedAt, &o.ExpiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("订单不存在")
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func scanOrders(rows *sql.Rows) ([]Order, error) {
	var out []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.ProductID, &o.ProductName, &o.PriceSnapshot, &o.Quantity, &o.TotalAmount, &o.Status, &o.CreatedAt, &o.ExpiredAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
