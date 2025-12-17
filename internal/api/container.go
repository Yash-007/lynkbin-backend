package api

import (
	"fmt"
	"module/lynkbin/internal/clients/gemini"
	"module/lynkbin/internal/db"
	"module/lynkbin/internal/middleware"
	"module/lynkbin/internal/repo"
	"module/lynkbin/internal/services/posts"
	"module/lynkbin/internal/services/users"
	"os"
)

type Container struct {
	MiddlewareService *middleware.MiddlewareService
	UserService       *users.UserService
	PostService       *posts.PostService
}

func NewContainer() *Container {
	// dbUrl := "postgresql://postgres:postgres@localhost:5432/lynkbin"
	dbUrl := os.Getenv("DB_URL")
	database := db.ConnectDB(dbUrl)

	// Run migrations
	// if err := db.MigrateDB(database); err != nil {
	// 	fmt.Printf("failed to migrate database: %v\n", err)
	// 	return nil
	// }

	geminiClient, err := gemini.NewGeminiClient("gemini-2.5-flash")
	if err != nil {
		fmt.Printf("failed to create gemini client: %v\n", err)
		return nil
	}
	postRepo := repo.NewPostRepo(database)
	userRepo := repo.NewUserRepo(database)

	middlewareService := middleware.NewMiddlewareService(userRepo)
	userService := users.NewUserService(userRepo)
	postService := posts.NewPostService(postRepo, geminiClient)

	return &Container{
		MiddlewareService: middlewareService,
		UserService:       userService,
		PostService:       postService,
	}
}
