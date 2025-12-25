package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"backend/internal/config"
)

var global *sql.DB

func Init(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn := cfg.DSN()
	if cfg.User == "" || cfg.Name == "" {
		return nil, fmt.Errorf("db config missing user or name")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	global = db
	return db, nil
}

func Get() *sql.DB {
	return global
}
