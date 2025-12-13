package models

import (
	"time"

	"github.com/lib/pq"
)

type Post struct {
	Id          int64          `json:"id" gorm:"primaryKey"`
	UserId      int64          `json:"user_id"`
	Data        string         `json:"data"`
	Platform    string         `json:"platform"`
	Author      string         `json:"author"`
	Category    string         `json:"category"`
	Topic       string         `json:"topic"`
	Tags        pq.StringArray `json:"tags" gorm:"type:text[]"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

type CreatePostResponse struct {
	Post
	PostLink string `json:"post_link"`
}

func (p Post) TableName() string {
	return "posts"
}
