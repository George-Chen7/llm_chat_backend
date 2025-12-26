package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/middlewares"
	"backend/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials 账号或密码错误。
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserNotFound 用户不存在。
	ErrUserNotFound = errors.New("user not found")
)

// Login 校验用户并生成 JWT。
func Login(ctx context.Context, username, password string) (string, error) {
	_, current, err := store.GetUserPassword(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotFound
		} else {
			return "", err
		}
	}

	ok, legacy, err := verifyPassword(current, password)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidCredentials
	}
	if legacy {
		if hashed, err := hashPassword(password); err == nil {
			_ = store.UpdateUserPassword(ctx, username, hashed)
		}
	}

	claims := middlewares.MyClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(middlewares.JWTSecret)
	return tokenString, nil
}

// EnsureAdmin 确保预置管理员存在。
func EnsureAdmin(ctx context.Context, cfg config.AdminConfig) error {
	if cfg.Username == "" || cfg.Password == "" {
		return nil
	}
	count, err := store.CountUsersByUsername(ctx, cfg.Username)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	nickname := cfg.Nickname
	if nickname == "" {
		nickname = cfg.Username
	}
	total := cfg.TotalQuota
	if total < 0 {
		total = 0
	}
	hashed, err := hashPassword(cfg.Password)
	if err != nil {
		return err
	}
	_, err = store.CreateUser(ctx, cfg.Username, hashed, nickname, "ADMIN", total, 0)
	return err
}

// ResetPassword 校验旧密码并更新为新密码。
func ResetPassword(ctx context.Context, username, oldPassword, newPassword string) error {
	_, current, err := store.GetUserPassword(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrUserNotFound
		}
		return err
	}
	ok, _, err := verifyPassword(current, oldPassword)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return store.UpdateUserPassword(ctx, username, hashed)
}

func hashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password required")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func verifyPassword(stored, password string) (bool, bool, error) {
	if isBcryptHash(stored) {
		err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password))
		if err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				return false, false, nil
			}
			return false, false, err
		}
		return true, false, nil
	}
	return stored == password, true, nil
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2") && len(value) >= 60
}

// RefreshToken 刷新 JWT。
func RefreshToken(username string) (string, error) {
	claims := middlewares.MyClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(middlewares.JWTSecret)
	return tokenString, nil
}
