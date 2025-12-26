package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"backend/internal/middlewares"
	"backend/internal/store"

	"github.com/golang-jwt/jwt/v5"
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
			role := "USER"
			total := int64(0)
			if username == "admin" {
				role = "ADMIN"
				total = 10000
			}
			if _, err := store.CreateUser(ctx, username, password, username, role, total, total); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	} else if current != password {
		return "", ErrInvalidCredentials
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

// ResetPassword 校验旧密码并更新为新密码。
func ResetPassword(ctx context.Context, username, oldPassword, newPassword string) error {
	_, current, err := store.GetUserPassword(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrUserNotFound
		}
		return err
	}
	if current != oldPassword {
		return ErrInvalidCredentials
	}
	return store.UpdateUserPassword(ctx, username, newPassword)
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
