package repository

import (
	"gorm.io/gorm"
)

type TestRepository struct {
	DB *gorm.DB
}

func NewTestRepository(db *gorm.DB) *TestRepository {
	return &TestRepository{DB: db}
}

func (r *TestRepository) PingDB() error {
	sqlDB, err := r.DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}