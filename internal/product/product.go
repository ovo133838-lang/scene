// Package product 商品域：上下架、改价、库存、可售数量查询。
package product

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/ovo133838-lang/scene/internal/store"
)

// Product 商品。
type Product struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Price  int64  `json:"price"`
	Stock  int    `json:"stock"`
	Status string `json:"status"`
}

// Service 商品域服务。
type Service struct {
	store *store.Store
}

// New 创建商品服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// ListOnSale 返回在售商品（买家可见）。
func (s *Service) ListOnSale() ([]Product, error) {
	return s.query(`SELECT id, name, price, stock, status FROM product WHERE status = 'on' ORDER BY id`)
}

// ListAll 返回全部商品（运营可见，含下架）。
func (s *Service) ListAll() ([]Product, error) {
	return s.query(`SELECT id, name, price, stock, status FROM product ORDER BY id`)
}

// StockBoard 返回库存看板（当前可售数量）。
func (s *Service) StockBoard() ([]Product, error) {
	return s.ListAll()
}

// Get 返回单个商品（不限状态）。
func (s *Service) Get(id int64) (*Product, error) {
	row := s.store.DB.QueryRow(`SELECT id, name, price, stock, status FROM product WHERE id = ?`, id)
	var p Product
	err := row.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("商品不存在")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SetShelf 上架/下架。
func (s *Service) SetShelf(id int64, status string) error {
	if status != "on" && status != "off" {
		return errors.New("非法状态")
	}
	res, err := s.store.DB.Exec(`UPDATE product SET status = ?, updated_at = NOW() WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("商品不存在")
	}
	return nil
}

// Update 改售价/库存（传入指针区分“未传”与“传 0”）。
func (s *Service) Update(id int64, price *int64, stock *int) error {
	var sets []string
	var args []interface{}
	if price != nil {
		if *price <= 0 {
			return errors.New("价格必须为正")
		}
		sets = append(sets, "price = ?")
		args = append(args, *price)
	}
	if stock != nil {
		if *stock < 0 {
			return errors.New("库存不能为负")
		}
		sets = append(sets, "stock = ?")
		args = append(args, *stock)
	}
	if len(sets) == 0 {
		return errors.New("无变更字段")
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)
	res, err := s.store.DB.Exec("UPDATE product SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("商品不存在")
	}
	return nil
}

func (s *Service) query(q string) ([]Product, error) {
	rows, err := s.store.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
