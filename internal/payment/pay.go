package payment

import (
	"fmt"
	"net/url"
	"time"
)

// PayResult 发起支付结果。
type PayResult struct {
	PayURL     string `json:"pay_url"`
	OutTradeNo string `json:"out_trade_no"`
}

// CreatePay 发起支付宝沙箱支付：校验订单可支付后，返回沙箱收银台链接。
// 私钥已配置则真实签名；未配置（教学环境）则返回未签名的演示链接，仍走同一业务规则。
func (s *Service) CreatePay(orderID string, buyerID int64) (*PayResult, error) {
	o, err := s.orders.PreparePay(orderID, buyerID)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"app_id":      s.cfg.Alipay.AppID,
		"method":      "alipay.trade.page.pay",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  s.cfg.Alipay.NotifyURL,
		"biz_content": buildBizContent(orderID, o.ProductName, o.TotalAmount),
	}

	if s.cfg.Alipay.PrivateKey != "" {
		sign, err := signRSA2(s.cfg.Alipay.PrivateKey, signSource(params))
		if err != nil {
			return nil, err
		}
		params["sign"] = sign
	}

	return &PayResult{
		PayURL:     s.cfg.Alipay.Gateway + "?" + encodeParams(params),
		OutTradeNo: orderID,
	}, nil
}

func buildBizContent(orderID, subject string, amountFen int64) string {
	return fmt.Sprintf(`{"out_trade_no":%q,"product_code":"FAST_INSTANT_TRADE_PAY","total_amount":%q,"subject":%q}`,
		orderID, fenToYuan(amountFen), subject)
}

// fenToYuan 分转元字符串（两位小数），避免浮点运算。
func fenToYuan(fen int64) string {
	return fmt.Sprintf("%d.%02d", fen/100, fen%100)
}

// encodeParams 按 key 字典序 URL 编码参数（url.Values.Encode 已排序）。
func encodeParams(params map[string]string) string {
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	return v.Encode()
}
