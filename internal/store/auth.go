package store

import "context"

// GetUserPassword 根据用户名获取用户ID与密码。
func GetUserPassword(ctx context.Context, username string) (int, string, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, "", err
	}
	var userID int
	var password string
	row := dbx.QueryRowContext(ctx, `
		SELECT user_id, password
		FROM users
		WHERE username = ? AND status = 1
	`, username)
	if err := row.Scan(&userID, &password); err != nil {
		return 0, "", err
	}
	return userID, password, nil
}

// GetUserByUsername 根据用户名获取用户信息（不含密码）。
func GetUserByUsername(ctx context.Context, username string) (User, error) {
	dbx, err := GetDB()
	if err != nil {
		return User{}, err
	}

	var u User
	row := dbx.QueryRowContext(ctx, `
		SELECT user_id, username, nickname, role, total_quota, remaining_quota
		FROM users
		WHERE username = ? AND status = 1
	`, username)
	if err := row.Scan(&u.UserID, &u.Username, &u.Nickname, &u.Role, &u.TotalQuota, &u.RemainingQuota); err != nil {
		return User{}, err
	}
	return u, nil
}

// CreateUser 创建用户并返回新ID。
func CreateUser(ctx context.Context, username, password, nickname, role string, total, remaining int64) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	res, err := dbx.ExecContext(ctx, `
		INSERT INTO users (username, password, nickname, role, status, total_quota, remaining_quota)
		VALUES (?, ?, ?, ?, 1, ?, ?)
	`, username, password, nickname, role, total, remaining)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}

// UpdateUserPassword 更新用户密码。
func UpdateUserPassword(ctx context.Context, username, newPassword string) error {
	dbx, err := GetDB()
	if err != nil {
		return err
	}
	_, err = dbx.ExecContext(ctx, `UPDATE users SET password = ? WHERE username = ?`, newPassword, username)
	return err
}

// CountUsersByUsername 统计用户名数量。
func CountUsersByUsername(ctx context.Context, username string) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	var count int
	if err := dbx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetUserRemainingQuota 获取用户剩余额度。
func GetUserRemainingQuota(ctx context.Context, userID int) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	var remaining int
	if err := dbx.QueryRowContext(ctx, `
		SELECT remaining_quota
		FROM users
		WHERE user_id = ? AND status = 1
	`, userID).Scan(&remaining); err != nil {
		return 0, err
	}
	return remaining, nil
}

// DecreaseUserQuota 扣减用户额度（允许为负）。
func DecreaseUserQuota(ctx context.Context, userID int, delta int) error {
	dbx, err := GetDB()
	if err != nil {
		return err
	}
	_, err = dbx.ExecContext(ctx, `
		UPDATE users
		SET remaining_quota = remaining_quota - ?
		WHERE user_id = ?
	`, delta, userID)
	return err
}
