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

	questionRepo := repository.NewQuestionRepository(db)
	questionService := service.NewQuestionService(questionRepo)
	questionHandler := handler.NewQuestionHandler(questionService)

	{
		r.GET("/health", testHandler.HealthCheck)

		//ユーザー作成のルーティング
		r.POST("/register", userhandler.Register)

		//問題のルーティング
		questionsGroup := r.Group("/questions")
		{
			questionsGroup.GET("", questionHandler.GetQuestions)
			questionsGroup.POST("", questionHandler.CreateQuestion)
		}
	}

	return r
}