package payment

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"sort"
	"strings"
)

// signSource 按支付宝规则生成待签名/验签源串：跳过 sign、sign_type 与空值，按 key 字典序拼接 key=value&...
func signSource(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(m[k])
	}
	return b.String()
}

// normalizePEM 归一化密钥字符串：还原被转义的换行，去除首尾空白。
// 约定 config 中存完整 PEM（含 BEGIN/END 头尾）。
func normalizePEM(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	return strings.TrimSpace(s)
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(pemStr)))
	if block == nil {
		return nil, errors.New("私钥 PEM 解析失败")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, errors.New("私钥不是 RSA 类型")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, errors.New("私钥解析失败（须为 PKCS1/PKCS8 PEM）")
}

func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(pemStr)))
	if block == nil {
		return nil, errors.New("公钥 PEM 解析失败")
	}
	if k, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PublicKey); ok {
			return rk, nil
		}
		return nil, errors.New("公钥不是 RSA 类型")
	}
	if k, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, errors.New("公钥解析失败（须为 PKIX/PKCS1 PEM）")
}

// signRSA2 使用商户私钥对源串做 SHA256withRSA 签名。
func signRSA2(privateKeyPEM, source string) (string, error) {
	key, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(source))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// verifyRSA2 使用支付宝公钥验签。
func verifyRSA2(publicKeyPEM, source, sign string) error {
	key, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return errors.New("签名 base64 解码失败")
	}
	h := sha256.Sum256([]byte(source))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, h[:], sig); err != nil {
		return errors.New("验签失败")
	}
	return nil
}
