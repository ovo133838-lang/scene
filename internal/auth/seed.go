package auth

import "golang.org/x/crypto/bcrypt"

// SeedUsers 用 bcrypt 初始化预置账号（密码=用户名），幂等可重复执行。
func (a *Auth) SeedUsers() error {
	users := []struct{ name, role string }{
		{"buyer1", "buyer"},
		{"buyer2", "buyer"},
		{"buyer3", "buyer"},
		{"operator", "operator"},
		{"warehouse", "warehouse"},
		{"service", "service"},
	}
	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.name), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := a.store.DB.Exec(
			`INSERT INTO user (username, password_hash, role) VALUES (?, ?, ?)
			 ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash), role = VALUES(role)`,
			u.name, string(hash), u.role,
		); err != nil {
			return err
		}
	}
	return nil
}
