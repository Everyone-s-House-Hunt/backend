package main

import (
	"log"
	"os"

	"house-hunt/router"
	"house-hunt/seed"
	"house-hunt/utils"
)

func main() {
	// 環境変数読み込み
	utils.LoadEnv()

	// DB接続
	db := utils.InitDB()
	if os.Getenv("SEED_BULLET_QUESTIONS") == "true" {
		if err := seed.EnsureBulletQuestions(db); err != nil {
			log.Fatalf("ゾンビバレット開発データの投入に失敗しました: %v", err)
		}
	}

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
