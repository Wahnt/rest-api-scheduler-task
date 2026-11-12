package auth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret   []byte
	password string
	ttl      time.Duration
	logger   *slog.Logger
}

func NewManager(secret string, password string, ttl time.Duration, logger *slog.Logger) *Manager {
	return &Manager{
		secret:   []byte(secret),
		password: password,
		ttl:      ttl,
		logger:   logger.With("component", "auth"),
	}
}

func (m *Manager) Login(password string) (string, error) {
	if password != m.password {
		m.logger.Warn("login failed: invalid password")
		return "", fmt.Errorf("invalid password")
	}

	claims := Claims{
		UserID: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		m.logger.Error("sign token failed", "err", err)
		return "", fmt.Errorf("sign token: %w", err)
	}
	m.logger.Info("login success")
	return signed, nil
}

func (m *Manager) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
