package models

import (
	"time"

	"github.com/lib/pq"
)

type UserCategories struct {
	UserId     int64          `json:"user_id" gorm:"primaryKey"`
	Platform   string         `json:"platform" gorm:"primaryKey"`
	Categories pq.StringArray `json:"categories" gorm:"type:text[]"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (u UserCategories) TableName() string {
	return "user_categories"
}

type AllCategories struct {
	Category  string    `json:"category" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (a AllCategories) TableName() string {
	return "all_categories"
}
