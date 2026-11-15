// Package payment 支付域：沙箱发起、异步通知验签与处理、幂等与迟到支付路由。
// 状态迁移集中在 order 包（MarkPaid/MarkPaymentAbnormal），本包只做验签、匹配与路由。
package payment

import (
	"github.com/ovo133838-lang/scene/internal/order"
	"github.com/ovo133838-lang/scene/internal/store"
)

// Service 支付域服务。
type Service struct {
	orders *order.Service
	cfg    *store.Config
}

// New 创建支付服务。
func New(o *order.Service, cfg *store.Config) *Service {
	return &Service{orders: o, cfg: cfg}
}
