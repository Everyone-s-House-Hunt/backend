package utils

import (
	"database/sql"
	"os"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func InitDB() *sql.DB {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)
	fmt.Printf("DSN: %s\n", dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("接続設定エラー: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("データベース接続エラー: %v", err)
	}
	fmt.Println("データベースに接続成功")
	return db
}