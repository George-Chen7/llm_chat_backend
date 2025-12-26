package store

import (
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/db"
)

// GetDB 获取全局数据库连接。
func GetDB() (*sql.DB, error) {
	dbx := db.Get()
	if dbx == nil {
		return nil, errors.New("db not initialized")
	}
	return dbx, nil
}

// BuildInClause 生成 IN 子句及其参数列表。
func BuildInClause(ids []int) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := make([]byte, 0, len(ids)*2)
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		args = append(args, id)
	}
	return fmt.Sprintf("(%s)", string(placeholders)), args
}
