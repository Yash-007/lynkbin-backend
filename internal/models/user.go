package models

import "time"

type User struct {
	Id              *int64    `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	Name            string    `json:"name"`
	Email           string    `json:"email" gorm:"unique;not null"`
	Password        string    `json:"password" gorm:"not null;"`
	TotalPosts      int64     `json:"total_posts" gorm:"default:0"`
	TotalCategories int64     `json:"total_categories" gorm:"default:0"`
	TotalTags       int64     `json:"total_tags" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (u User) TableName() string {
	return "users"
}
