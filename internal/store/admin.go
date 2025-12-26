package store

import (
	"context"
)

// ListUsers 分页获取用户列表。
func ListUsers(ctx context.Context, page, pageSize int) ([]User, int, error) {
	dbx, err := GetDB()
	if err != nil {
		return nil, 0, err
	}
	var totalCount int
	if err := dbx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&totalCount); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	rows, err := dbx.QueryContext(ctx, `
		SELECT user_id, username, nickname, role, total_quota, remaining_quota
		FROM users
		ORDER BY user_id DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.UserID, &u.Username, &u.Nickname, &u.Role, &u.TotalQuota, &u.RemainingQuota); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, totalCount, nil
}

// CreateUserWithQuota 创建用户并返回对象。
func CreateUserWithQuota(ctx context.Context, username, password, nickname, role string, total, remaining int) (User, error) {
	newID, err := CreateUser(ctx, username, password, nickname, role, int64(total), int64(remaining))
	if err != nil {
		return User{}, err
	}
	return User{
		UserID:         newID,
		Username:       username,
		Nickname:       nickname,
		Role:           role,
		TotalQuota:     total,
		RemainingQuota: remaining,
	}, nil
}

// SetUserQuota 设置用户额度，返回是否命中。
func SetUserQuota(ctx context.Context, userID int, quota int) (bool, error) {
	dbx, err := GetDB()
	if err != nil {
		return false, err
	}
	res, err := dbx.ExecContext(ctx, `
		UPDATE users SET total_quota = ?, remaining_quota = ?
		WHERE user_id = ?
	`, quota, quota, userID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// DeleteUser 删除用户，返回是否命中。
func DeleteUser(ctx context.Context, userID int) (bool, error) {
	dbx, err := GetDB()
	if err != nil {
		return false, err
	}
	res, err := dbx.ExecContext(ctx, `DELETE FROM users WHERE user_id = ?`, userID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}
