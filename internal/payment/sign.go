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

// keyDER 提取密钥 DER 字节：支持完整 PEM（含 BEGIN/END）与裸 base64（支付宝密钥工具格式）。
func keyDER(pemStr string) ([]byte, error) {
	s := strings.ReplaceAll(pemStr, `\n`, "\n")
	s = strings.TrimSpace(s)
	if strings.Contains(s, "-----BEGIN") {
		block, _ := pem.Decode([]byte(s))
		if block == nil {
			return nil, errors.New("密钥 PEM 解析失败")
		}
		return block.Bytes, nil
	}
	// 裸 base64（可能含换行/空白），去空白后解码
	return base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), ""))
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	der, err := keyDER(pemStr)
	if err != nil {
		return nil, err
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, errors.New("私钥不是 RSA 类型")
	}
	if k, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return k, nil
	}
	return nil, errors.New("私钥解析失败（须为 PKCS1/PKCS8）")
}

func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	der, err := keyDER(pemStr)
	if err != nil {
		return nil, err
	}
	if k, err := x509.ParsePKIXPublicKey(der); err == nil {
		if rk, ok := k.(*rsa.PublicKey); ok {
			return rk, nil
		}
		return nil, errors.New("公钥不是 RSA 类型")
	}
	if k, err := x509.ParsePKCS1PublicKey(der); err == nil {
		return k, nil
	}
	return nil, errors.New("公钥解析失败（须为 PKIX/PKCS1）")
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
