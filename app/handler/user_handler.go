package handler

import (
	"house-hunt/service"
	"house-hunt/dto"
	"house-hunt/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	Service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{Service: service}
}

// ユーザー登録
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "Error",
			"message": "Invalid request data: " + err.Error(),
		})
		return
	}

	user, err := h.Service.Register(req.Username, req.Email, req.Password)
	if err != nil {
		if err == utils.ErrDuplicateEmail {
			c.JSON(http.StatusConflict, gin.H{
				"status": "Error",
				"message": err.Error(),
			})
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "Error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "OK",
		"message": "User registered successfully",
		"user": gin.H{
			"id": user.ID,
			"username": user.Username,
			"is_premium": user.IsPremium,
			"created_at": user.CreatedAt,
		},
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "Error",
			"message": utils.ErrInvalidInput.Error(),
		})
		return
	}

	accessToken, refreshToken, err := h.Service.Login(req)
	if err != nil {
		if err == utils.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "Error",
				"message": utils.ErrInvalidCredentials,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error",
				"message": utils.ErrInternalServer,
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "OK",
		"access_token": accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *UserHandler) LoginWithGoogle(c *gin.Context) {
	var req dto.GoogleLoginRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "Error",
			"message": utils.ErrInvalidInput,
		})
		return
	}

	accessToken, refreshToken, err := h.Service.LoginWithGoogle(c.Request.Context(), req.IDToken)
	if err != nil {
		if err == utils.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "Error",
				"message": utils.ErrInvalidCredentials,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "Error",
				"message": utils.ErrInternalServer,
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "OK",
		"access_token": accessToken,
		"refresh_token": refreshToken,
	})
}