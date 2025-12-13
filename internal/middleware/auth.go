package middleware

import (
	"module/lynkbin/internal/repo"
	"module/lynkbin/internal/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type MiddlewareService struct {
	userRepo *repo.UserRepo
}

func NewMiddlewareService(userRepo *repo.UserRepo) *MiddlewareService {
	return &MiddlewareService{userRepo: userRepo}
}

func (m *MiddlewareService) AuthMiddleware(ctx *gin.Context) {
	platform := ctx.Request.Header.Get("X-Platform-Id")
	if platform == "telegram-bot" {
		email := ctx.Request.Header.Get("X-Email-Id")
		if email == "" {
			utilities.Response(ctx, http.StatusUnauthorized, false, nil, "Unauthorized")
			ctx.Abort()
			return
		}
		userId, err := m.userRepo.GetUserIdByEmail(email)
		if err != nil {
			utilities.Response(ctx, http.StatusUnauthorized, false, nil, "Unauthorized")
		}
		ctx.Set("user_id", userId)
		ctx.Next()
		return
	}
	authToken := ctx.Request.Header.Get("X-Auth-Token")
	if authToken == "" {
		utilities.Response(ctx, http.StatusUnauthorized, false, nil, "Unauthorized")
		ctx.Abort()
		return
	}

	token, err := jwt.ParseWithClaims(authToken, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return []byte("secret_key"), nil
	})

	if err != nil {
		utilities.Response(ctx, http.StatusUnauthorized, false, nil, "Unauthorized")
		ctx.Abort()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		utilities.Response(ctx, http.StatusUnauthorized, false, nil, "Unauthorized")
		ctx.Abort()
		return
	}

	userId := claims["user_id"]
	ctx.Set("user_id", int64(userId.(float64)))
	ctx.Next()
}
