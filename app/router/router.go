package router

import (
	"gorm.io/gorm"
	"house-hunt/handler"
	"house-hunt/repository"
	"house-hunt/service"

	"github.com/gin-gonic/gin"
)


func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	testRepo := repository.NewTestRepository(db)
	testService := service.NewTestService(testRepo)
	testHandler := handler.NewTestHandler(testService)

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userhandler := handler.NewUserHandler(userService)

	// WebSocket 配線。RoomManager は全体で1つだけ生成し共有する。
	questionRepo := repository.NewQuestionRepository(db)
	roomManager := service.NewRoomManager(questionRepo)
	wsHandler := handler.NewWSHandler(roomManager)

	// ルーティングの設定
	r.GET("/health", testHandler.HealthCheck)

	r.POST("/register", userhandler.Register)

	// :roomID はフロントが生成した6桁ルームID
	r.GET("/ws/rooms/:roomID", wsHandler.Connect)

	return r
}