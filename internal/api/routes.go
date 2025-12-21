package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, container *Container) {
	middlewareService := container.MiddlewareService

	userRoutes := router.Group("/users")

	userRoutes.GET("/me", middlewareService.AuthMiddleware, container.UserService.GetCurrentUser)
	userRoutes.POST("/register", container.UserService.RegisterUser)
	userRoutes.POST("/login", container.UserService.LoginUser)

	postRoutes := router.Group("/posts")
	postRoutes.POST("", middlewareService.AuthMiddleware, container.PostService.CreatePost)
	postRoutes.GET("", middlewareService.AuthMiddleware, container.PostService.GetPosts)
	postRoutes.DELETE("/:id", middlewareService.AuthMiddleware, container.PostService.DeletePost)
	postRoutes.GET("/authors", middlewareService.AuthMiddleware, container.PostService.GetUserAuthors)
	postRoutes.GET("/categories", middlewareService.AuthMiddleware, container.PostService.GetUserCategories)
	postRoutes.GET("/tags", middlewareService.AuthMiddleware, container.PostService.GetUserTags)

	postRoutes.GET("/counts", middlewareService.AuthMiddleware, container.PostService.GetAllUserPostsTagsAndCategoriesCount)
}
