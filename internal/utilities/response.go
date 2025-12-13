package utilities

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type GenericResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

func Response(ctx *gin.Context, statusCode int, success bool, data interface{}, message string) {
	response := GenericResponse{
		Success: success,
		Data:    data,
		Message: message,
	}

	ctx.JSON(statusCode, response)
}

type CustomClaims struct {
	UserId int64 `json:"user_id"`
	jwt.RegisteredClaims
}
