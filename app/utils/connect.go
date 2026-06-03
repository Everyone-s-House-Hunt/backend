package utils

import (
	"os"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"house-hunt/model"
)

func InitDB() *gorm.DB {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Asia%%2FTokyo", user, password, host, port, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("接続設定エラー: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Subscription{},
		&model.Question{},
	)
	if err != nil {
		log.Fatalf("マイグレーションエラー: %v", err)
	}

	fmt.Println("データベース接続成功")
	return db
}