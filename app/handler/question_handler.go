package handler

import (
	"house-hunt/dto"
	"house-hunt/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QestionHandler struct {
	Service *service.QuestionService
}

func NewQuestionHandler(service *service.QuestionService) *QestionHandler {
	return &QestionHandler{Service: service}
}

func (h *QestionHandler) GetQuestions(c *gin.Context) {
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