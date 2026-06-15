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

	// WebSocket 配線。RoomManager は全体で1つだけ生成し共有する。
	roomManager := service.NewRoomManager(questionRepo)
	wsHandler := handler.NewWSHandler(roomManager)

	{
		r.GET("/health", testHandler.HealthCheck)

		//ユーザー作成のルーティング
		r.POST("/register", userhandler.Register)

		//問題のルーティング
		questionsGroup := r.Group("/questions")
		{
			// ゲームモードごとの問題取得
			questionsGroup.GET("", questionHandler.GetQuestions)
			// 問題の作成
			questionsGroup.POST("", questionHandler.CreateQuestion)

			// 問題の承認ステータス更新
			questionsGroup.PATCH("/:id/status", questionHandler.UpdateQuestionStatus)
		}

		// :roomID はフロントが生成した6桁ルームID
		r.GET("/ws/rooms/:roomID", wsHandler.Connect)
	}

	return r
}
