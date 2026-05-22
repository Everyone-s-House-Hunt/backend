package handler

import (
	"house-hunt/service"

	"net/http"

	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	Service *service.TestService
}

func NewTestHandler(service *service.TestService) *TestHandler {
	return &TestHandler{Service: service}
}

func (h *TestHandler) HealthCheck(c *gin.Context) {
	err := h.Service.PingDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "Error",
			"message": err.Error(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "OK",
	})
}