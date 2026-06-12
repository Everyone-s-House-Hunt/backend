package handler

import (
	"house-hunt/dto"
	"house-hunt/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QuestionHandler struct {
	Service *service.QuestionService
}

func NewQuestionHandler(service *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{Service: service}
}

func (h *QuestionHandler) GetQuestions(c *gin.Context) {
	gameMode := c.Query("game_mode")
	if gameMode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "Error",
			"message": "game_mode is required",
		})
		return
	}

	limitStr := c.Query("limit")
	limit, _ := strconv.Atoi(limitStr)

	questions, err := h.Service.GetQuestions(gameMode, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "Error",
			"message": err.Error(),
		})
		return
	}

	var responseData []dto.QuestionResponse
	for _, q := range questions {
		responseData = append(responseData, dto.BuildQuestionResponse(q))
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "OK",
		"count": len(questions),
		"data": responseData,
	})
}

func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	var req dto.CreateQuestionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "Error",
			"message": "リクエストデータの形式に不備があります",
			"details": err.Error(),
		})
		return
	}

	if req.CorrectIndex >= len(req.Choices) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "Error",
			"message": "correct_index は choices の要素数未満である必要があります",
		})
		return
	}

	question, err := h.Service.CreateQuestion(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "Error",
			"message": "問題の登録に失敗しました",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "Created",
		"data":   dto.BuildQuestionResponse(*question),
	})
}