package models

import (
	"time"

	"github.com/lib/pq"
)

type UserTags struct {
	UserId    int64          `json:"user_id" gorm:"primaryKey"`
	Platform  string         `json:"platform" gorm:"primaryKey"`
	Tags      pq.StringArray `json:"tags" gorm:"type:text[]"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (u UserTags) TableName() string {
	return "user_tags"
}

type AllTags struct {
	Tag       string    `json:"tag" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (a AllTags) TableName() string {
	return "all_tags"
}
