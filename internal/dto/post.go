package dto

import "github.com/lib/pq"

type CreatePostRequest struct {
	Url   string   `json:"url" validate:"url"`
	Notes string   `json:"notes"`
	IsUrl bool     `json:"is_url"`
	Tags  []string `json:"tags"`
}

type SummarizePostResponse struct {
	Category    string         `json:"category"`
	Topic       string         `json:"topic"`
	Tags        pq.StringArray `json:"tags"`
	Description string         `json:"description"`
}

type GetPostsRequest struct {
	Platform   string   `form:"platform"`
	Tags       []string `form:"tags"`
	Authors    []string `form:"authors"`
	Categories []string `form:"categories"`
}

type GetAllTagsAndCategoriesCountResponse struct {
	TotalPostsCount      int64 `json:"total_posts_count"`
	TotalTagsCount       int64 `json:"total_tags_count"`
	TotalCategoriesCount int64 `json:"total_categories_count"`
}

type Media struct {
	Path    string `json:"path"`
	Context string `json:"context"`
}

type MediaData struct {
	IsMedia bool    `json:"is_media"`
	Media   []Media `json:"media"`
}
