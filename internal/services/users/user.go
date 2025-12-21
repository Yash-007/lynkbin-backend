package users

import (
	"errors"
	"fmt"
	"log"
	"module/lynkbin/internal/dto"
	"module/lynkbin/internal/models"
	"module/lynkbin/internal/repo"
	"module/lynkbin/internal/utilities"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *repo.UserRepo
}

func NewUserService(userRepo *repo.UserRepo) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) RegisterUser(ctx *gin.Context) {
	var request dto.RegisterUserRequest
	err := ctx.ShouldBindBodyWithJSON(&request)
	if err != nil {
		utilities.Response(ctx, 400, false, nil, "Invalid request body")
		return
	}
	validator := validator.New()
	if err := validator.Struct(request); err != nil {
		fmt.Printf("Invalid request body: %v\n", err)
		utilities.Response(ctx, 400, false, nil, "Invalid request body")
		return
	}
	user, err := s.userRepo.GetUserByEmail(request.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error getting user by email: %v\n", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get user by email")
		return
	}
	if user != nil {
		utilities.Response(ctx, 400, false, nil, "User already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), 10)
	if err != nil {
		fmt.Printf("Error hashing password: %v\n", err)
		utilities.Response(ctx, 500, false, nil, "Internal server error")
		return
	}

	newUser := models.User{
		Name:     request.Name,
		Email:    request.Email,
		Password: string(hashedPassword),
	}

	err = s.userRepo.CreateUser(&newUser)
	if err != nil {
		log.Printf("Error creating user: %v\n", err)
		utilities.Response(ctx, 500, false, nil, "Failed to create user")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(*newUser.Id),
		"exp":     time.Now().Add(time.Hour * 720).Unix(),
	})

	signedToken, err := token.SignedString([]byte("secret_key"))
	if err != nil {
		log.Printf("Error signing token: %v\n", err)
		utilities.Response(ctx, 500, false, nil, "Failed to generate token")
		return
	}

	utilities.Response(ctx, 201, true, gin.H{"token": signedToken}, "User registered successfully")
}

func (s *UserService) LoginUser(ctx *gin.Context) {
	var request dto.LoginUserRequest
	err := ctx.ShouldBindBodyWithJSON(&request)
	if err != nil {
		fmt.Println("Error binding request body: ", err)
		utilities.Response(ctx, 400, false, nil, "Invalid request body")
		return
	}
	validator := validator.New()
	if err := validator.Struct(request); err != nil {
		fmt.Printf("Invalid request body: %v\n", err)
		utilities.Response(ctx, 400, false, nil, "Invalid request body")
		return
	}
	user, err := s.userRepo.GetUserByEmail(request.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error getting user by email: %v\n", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get user by email")
		return
	}
	if user == nil {
		utilities.Response(ctx, 400, false, nil, "User not found")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password))
	if err != nil {
		fmt.Println("Error comparing hash and password: ", err)
		utilities.Response(ctx, 400, false, nil, "Invalid password")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(*user.Id),
		"exp":     time.Now().Add(time.Hour * 720).Unix(),
	})

	signedToken, err := token.SignedString([]byte("secret_key"))
	if err != nil {
		log.Printf("Error signing token: %v\n", err)
		utilities.Response(ctx, 500, false, nil, "Failed to generate token")
		return
	}

	utilities.Response(ctx, 200, true, gin.H{"token": signedToken}, "User logged in successfully")
}

func (s *UserService) GetCurrentUser(ctx *gin.Context) {
	userId, ok := ctx.Get("user_id")
	if !ok {
		utilities.Response(ctx, 400, false, nil, "Invalid user id")
		return
	}

	user, err := s.userRepo.GetUserById(userId.(int64))
	user.Password = ""
	if err != nil {
		fmt.Println("Error getting user by id: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to fetch user")
		return
	}
	utilities.Response(ctx, 200, true, user, "User fetched successfully")
}
