package handler

import (
	"house-hunt/dto"
	"house-hunt/service"
	"net/http"
	"strconv"

	"house-hunt/utils"

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
			"status":  "Error",
			"message": utils.ErrInvalidInput.Error(),
		})
		return
	}

	limitStr := c.Query("limit")
	limit, _ := strconv.Atoi(limitStr)

	questions, err := h.Service.GetQuestions(gameMode, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "Error",
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
		"count":  len(questions),
		"data":   responseData,
	})
}

func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	var req dto.CreateQuestionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "Error",
			"message": utils.ErrInvalidInput.Error(),
			"details": err.Error(),
		})
		return
	}

	question, err := h.Service.CreateQuestion(req)
	if err != nil {
		if service.IsInvalidQuestion(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "Error",
				"message": utils.ErrInvalidInput.Error(),
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "Error",
			"message": utils.ErrDatabase.Error(),
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "Created",
		"data":   dto.BuildQuestionResponse(*question),
	})
}

func (h *QuestionHandler) UpdateQuestionStatus(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateQuestionStatusRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "Error",
			"message": utils.ErrInvalidInput.Error(),
			"details": err.Error(),
		})
		return
	}

	if err := h.Service.UpdateQuestionStatus(id, req); err != nil {
		if err == utils.ErrNotFoundID {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "Error",
				"message": err,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "Error",
			"message": err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "OK",
		"message": "問題のステータスを更新しました",
	})
}
