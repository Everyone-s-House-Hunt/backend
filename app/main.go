package main

import (
	"house-hunt/router"
	"house-hunt/utils"
)

func main() {
	// 環境変数読み込み
	utils.LoadEnv()

	// DB接続
	db := utils.InitDB()
	defer db.Close()

	r := router.SetupRouter(db)

	r.Run(":8080")
}