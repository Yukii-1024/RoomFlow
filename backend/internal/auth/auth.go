// Package auth issues and validates JWTs and hashes passwords.
package auth

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const issuer = "roomflow"

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

// Issue returns the signed token and its lifetime in seconds.
func (m *Manager) Issue(userID int64, role string) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}
	return signed, int(m.ttl.Seconds()), nil
}

func (m *Manager) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if _, err := strconv.ParseInt(claims.Subject, 10, 64); err != nil {
		return nil, errors.New("invalid subject")
	}
	return claims, nil
}

// UserID returns the numeric subject of a parsed token.
func (c *Claims) UserID() int64 {
	id, _ := strconv.ParseInt(c.Subject, 10, 64)
	return id
}

const (
	MinPasswordLength = 8
	MaxPasswordLength = 72 // bcrypt input limit
)

// ValidatePassword enforces the shared password policy.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("密码长度不能少于 %d 位", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("密码长度不能超过 %d 位", MaxPasswordLength)
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("密码不能为空白")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
