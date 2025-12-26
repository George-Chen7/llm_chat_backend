package service

import (
	"context"

	"backend/internal/store"
)

// ListUsers 分页获取用户列表。
func ListUsers(ctx context.Context, page, pageSize int) ([]store.User, int, error) {
	return store.ListUsers(ctx, page, pageSize)
}

// CreateUser 创建用户并返回对象。
func CreateUser(ctx context.Context, username, password, nickname, role string, total, remaining int) (store.User, error) {
	hashed, err := hashPassword(password)
	if err != nil {
		return store.User{}, err
	}
	return store.CreateUserWithQuota(ctx, username, hashed, nickname, role, total, remaining)
}

// SetUserQuota 设置用户额度。
func SetUserQuota(ctx context.Context, userID int, quota int) (bool, error) {
	return store.SetUserQuota(ctx, userID, quota)
}

// DeleteUser 删除用户。
func DeleteUser(ctx context.Context, userID int) (bool, error) {
	return store.DeleteUser(ctx, userID)
}
