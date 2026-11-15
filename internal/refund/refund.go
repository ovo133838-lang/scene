// Package refund 退款域：售前退款申请/审核/渠道退款/重试，迟到支付退款（不恢复库存）。
// 订单状态迁移经 order.Transition 复用同一状态机；本包管理 refund / payment_abnormal 表。
package refund

import (
	"errors"
	"time"

	"github.com/ovo133838-lang/scene/internal/order"
	"github.com/ovo133838-lang/scene/internal/store"
)

// 退款单状态（与 docs/02-数据字典.md 一致，另补 rejected 记录被拒的退款）。
const (
	StatusPending    = "pending"    // 待审核
	StatusProcessing = "processing" // 渠道退款中
	StatusSuccess    = "success"    // 退款成功
	StatusFailed     = "failed"     // 退款失败（可重试）
	StatusRejected   = "rejected"   // 已拒绝
)

// 退款类型。
const (
	TypeSale         = "sale"          // 售前退款（成功后恢复库存）
	TypeLatePayment  = "late_payment"  // 迟到支付退款（不恢复库存）
)

// 哨兵错误。
var (
	ErrRefundNotFound   = errors.New("退款单不存在")
	ErrRefundState      = errors.New("退款单状态不允许该操作")
	ErrAbnormalNotFound = errors.New("异常支付记录不存在")
	ErrAbnormalState    = errors.New("异常支付状态不允许该操作")
	ErrReasonRequired  = errors.New("拒绝退款须填写原因")
	ErrNotRefundable   = errors.New("当前状态不可申请退款")
	ErrDuplicateRefund = errors.New("该订单已有待处理退款")
)

// Refund 退款单（查询用）。
type Refund struct {
	ID         int64     `json:"id"`
	OrderID    string    `json:"order_id"`
	PaymentID  *int64    `json:"payment_id"`
	Amount     int64     `json:"amount"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	Reason     string    `json:"reason"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Abnormal 异常支付记录（查询用）。
type Abnormal struct {
	ID            int64     `json:"id"`
	OrderID       string    `json:"order_id"`
	ChannelTxnNo  string    `json:"channel_txn_no"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
}

// Service 退款域服务。
type Service struct {
	store  *store.Store
	orders *order.Service
	cfg    *store.Config
	// channelRefund 渠道退款模拟；nil 时用 simulateChannelRefund。测试可注入。
	channelRefund func(orderID string, amount int64, retryCount int) error
}

// New 创建退款服务。
func New(s *store.Store, o *order.Service, cfg *store.Config) *Service {
	return &Service{store: s, orders: o, cfg: cfg}
}

// simulateChannelRefund 默认渠道退款模拟：成功；配置 fail_first_attempt 时首次失败。
func (s *Service) simulateChannelRefund(orderID string, amount int64, retryCount int) error {
	if s.cfg.Refund.FailFirstAttempt && retryCount == 0 {
		return errors.New("渠道退款失败（模拟）")
	}
	return nil
}

func (s *Service) doChannelRefund(orderID string, amount int64, retryCount int) error {
	if s.channelRefund != nil {
		return s.channelRefund(orderID, amount, retryCount)
	}
	return s.simulateChannelRefund(orderID, amount, retryCount)
}
