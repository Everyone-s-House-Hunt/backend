package repository

import (
	"database/sql"
)

type TestRepository struct {
	DB *sql.DB
}

func NewTestRepository(db *sql.DB) *TestRepository {
	return &TestRepository{DB: db}
}

func (r *TestRepository) PingDB()  error {
	return r.DB.Ping()
}