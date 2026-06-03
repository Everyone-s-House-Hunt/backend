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

	// ルーティングの設定
	r.GET("/health", testHandler.HealthCheck)

	r.POST("/register", userhandler.Register)

	return r
}