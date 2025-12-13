package db

import (
	"fmt"
	"module/lynkbin/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB(dbURL string) *gorm.DB {
	dbObj, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		fmt.Printf("getting error while connecting to DB: %v\n", err)
	}

	fmt.Println("Database connection established")
	return dbObj
}

func MigrateDB(db *gorm.DB) error {
	fmt.Println("Running database migrations...")

	// Add all your models here to auto-migrate
	err := db.AutoMigrate(
		&models.Post{},
		&models.User{},
		&models.UserAuthor{},
		&models.UserTags{},
		&models.AllTags{},
		&models.UserCategories{},
		&models.AllCategories{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("Database migrations completed successfully")
	return nil
}
