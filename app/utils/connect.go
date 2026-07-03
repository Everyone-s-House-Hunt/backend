package utils

import (
	"os"
	"fmt"
	"log"
	"time"

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

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Asia%%2FTokyo", user, password, host, port, dbname)

	// コンテナ起動直後は MySQL の準備が終わっていないことがあるため、リトライして待つ
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 10; attempt++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("DB接続待機中 (%d/10): %v", attempt, err)
		time.Sleep(3 * time.Second)
	}
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