package main

import (
	"os"
	"log"

	"house-hunt/router"
	"house-hunt/utils"
)

func main() {
	// 環境変数読み込み
	utils.LoadEnv()

	// DB接続
	db := utils.InitDB()

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("GORMからsql.DBの取得に失敗しました: %v", err)
	}
	defer sqlDB.Close()

	r := router.SetupRouter(db)

	r.Run(":" + os.Getenv("GO_PORT"))
	if err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}