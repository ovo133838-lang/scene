// Package clock 提供可配置的时间源，用于待付款超时等时间边界的统一判断。
// 测试与演示共用同一业务规则；教学演示可缩短时长，须注明。
package clock

import "time"

// Clock 是时间源。后续超时判断统一通过 Clock.Now() 取服务端时间。
type Clock struct {
	now func() time.Time
}

// New 返回使用真实时间的时间源。
func New() *Clock {
	return &Clock{now: time.Now}
}

// Now 返回当前服务端时间。
func (c *Clock) Now() time.Time {
	return c.now()
}
