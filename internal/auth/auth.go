// Package auth 提供 JWT 鉴权与 RBAC 角色守卫。
// 身份一律取自 JWT，不信任请求参数中的用户编号（§2.3 鉴权硬性要求）。
package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/ovo133838-lang/scene/internal/store"
)

// Auth 封装鉴权能力。
type Auth struct {
	store  *store.Store
	secret []byte
}

// New 创建 Auth。
func New(s *store.Store, secret string) *Auth {
	return &Auth{store: s, secret: []byte(secret)}
}

// Login 校验用户名密码，返回 JWT、角色、用户 ID。
func (a *Auth) Login(username, password string) (token string, role string, userID int64, err error) {
	var hash string
	err = a.store.DB.QueryRow(
		`SELECT id, password_hash, role FROM user WHERE username = ?`, username,
	).Scan(&userID, &hash, &role)
	if err != nil {
		return "", "", 0, errors.New("用户名或密码错误")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", "", 0, errors.New("用户名或密码错误")
	}
	token, err = a.generateToken(userID, role)
	return token, role, userID, err
}

func (a *Auth) generateToken(userID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// Middleware 校验 Bearer Token，将 user_id 与 role 写入上下文。
func (a *Auth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"code": 40100, "message": "未认证"})
			return
		}
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(strings.TrimPrefix(h, "Bearer "), claims, func(t *jwt.Token) (interface{}, error) {
			return a.secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"code": 40100, "message": "token 无效或过期"})
			return
		}
		uid, _ := claims["user_id"].(float64)
		role, _ := claims["role"].(string)
		c.Set("user_id", int64(uid))
		c.Set("role", role)
		c.Next()
	}
}

// RequireRole 要求当前角色属于 roles 之一，否则 403。
func (a *Auth) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(403, gin.H{"code": 40300, "message": "无权限"})
	}
}

// UserID 从上下文取当前用户 ID。
func UserID(c *gin.Context) int64 {
	return c.GetInt64("user_id")
}

// Role 从上下文取当前角色。
func Role(c *gin.Context) string {
	return c.GetString("role")
}

// LoginHandler 处理 POST /api/login。
func (a *Auth) LoginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(400, gin.H{"code": 40000, "message": "参数错误"})
		return
	}
	token, role, id, err := a.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"code": 40100, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{
		"token": token, "role": role, "user_id": id,
	}})
}
