package models

import (
	"time"

	"github.com/lib/pq"
)

type UserAuthor struct {
	UserId    int64          `json:"user_id" gorm:"primaryKey"`
	Platform  string         `json:"platform" gorm:"primaryKey"`
	Names     pq.StringArray `json:"names" gorm:"type:text[]"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

func (u UserAuthor) TableName() string {
	return "user_authors"
}
