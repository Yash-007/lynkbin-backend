package posts

import (
	"context"
	"encoding/json"
	"fmt"
	"module/lynkbin/internal/clients/gemini"
	"module/lynkbin/internal/dto"
	"module/lynkbin/internal/models"
	"module/lynkbin/internal/repo"
	"module/lynkbin/internal/scraper"
	"module/lynkbin/internal/utilities"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"
)

type PostService struct {
	postRepo     *repo.PostRepo
	geminiClient *gemini.GeminiClient
}

func NewPostService(postRepo *repo.PostRepo, geminiClient *gemini.GeminiClient) *PostService {
	return &PostService{postRepo: postRepo, geminiClient: geminiClient}
}

func (s *PostService) ExtractPostPlatform(userPost string, isUrl bool) (string, error) {
	if !isUrl {
		return "notes", nil
	}
	if strings.Contains(userPost, "linkedin.com") {
		return "linkedin", nil
	} else if strings.Contains(userPost, "x.com") {
		return "x", nil
	} else if strings.Contains(userPost, "reddit.com") {
		return "reddit", nil
	} else if strings.Contains(userPost, "instagram.com") {
		return "instagram", nil
	} else {
		return "others", nil
	}
}

func (s *PostService) GenerateTagsAndCategoriesData(userTags []string) (string, string, string, error) {
	allTags, err := s.postRepo.GetAllTags()
	if err != nil {
		fmt.Println("Error getting all tags: ", err)
		return "", "", "", err
	}
	tags := make([]string, len(allTags))
	for i, tag := range allTags {
		tags[i] = tag.Tag
	}
	tagString := strings.Join(tags, ", ")

	allCategories, err := s.postRepo.GetAllCategories()
	if err != nil {
		fmt.Println("Error getting all categories: ", err)
		return "", "", "", err
	}
	categories := make([]string, len(allCategories))
	for i, category := range allCategories {
		categories[i] = category.Category
	}
	categoryString := strings.Join(categories, ", ")

	userTagsString := strings.Join(userTags, ", ")
	return tagString, categoryString, userTagsString, nil
}

func (s *PostService) SummarizePost(ctx *gin.Context, content string, userTags []string, MediaData dto.MediaData) (dto.SummarizePostResponse, error) {
	tagString, categoryString, userTagsString, err := s.GenerateTagsAndCategoriesData(userTags)
	if err != nil {
		fmt.Println("Error generating tags and categories data: ", err)
		return dto.SummarizePostResponse{}, err
	}
	prompt := ""
	summary := ""

	if MediaData.IsMedia {
		prompt = utilities.GenerateMediaCategorizationPrompt(MediaData.Media, tagString, categoryString, userTagsString)
		summary, err = s.geminiClient.GenerateContentWithMedia(context.Background(), prompt, MediaData.Media)

	} else {
		prompt = utilities.GenerateCategorizationPrompt(content, tagString, categoryString, userTagsString)
		summary, err = s.geminiClient.GenerateContent(context.Background(), prompt)
	}

	if err != nil {
		fmt.Println("Error generating content: ", err)
		return dto.SummarizePostResponse{}, err
	}
	summary = utilities.CleanJSONResponse(summary)

	fmt.Println("Cleaned JSON response:", summary)

	var summaryJson dto.SummarizePostResponse
	err = json.Unmarshal([]byte(summary), &summaryJson)
	if err != nil {
		fmt.Println("Error unmarshalling summary:", err)
		fmt.Println("Raw response:", summary)
		return dto.SummarizePostResponse{}, err
	}
	return summaryJson, nil
}

func (s *PostService) ExtractPostDetails(ctx *gin.Context, userPost string, platform string, tags []string) (models.Post, error) {
	var summary dto.SummarizePostResponse
	var scrapedPost scraper.ScrapedPost
	author := ""
	var err error
	if platform == "linkedin" {
		// scrapedPost, err = scraper.ScrapeLinkedInPost(userPost, "socks5://10.101.116.69:1088")
		scrapedPost, err = scraper.ScrapeLinkedInPost(userPost, "")
		if err != nil {
			fmt.Println("Error scraping LinkedIn post: ", err)
			return models.Post{}, err
		}
		author = scrapedPost.Author
		summary, err = s.SummarizePost(ctx, scrapedPost.Content, tags, dto.MediaData{IsMedia: false, Media: nil})
		if err != nil {
			fmt.Println("Error summarizing LinkedIn post: ", err)
			return models.Post{}, err
		}
	} else if platform == "x" {
		// scrapedPost, err = scraper.ScrapeXPost(userPost, "socks5://10.101.116.69:1088")
		scrapedPost, err = scraper.ScrapeXPost(userPost, "")
		if err != nil {
			fmt.Println("Error scraping X post: ", err)
			return models.Post{}, err
		}
		author = scrapedPost.Author
		summary, err = s.SummarizePost(ctx, scrapedPost.Content, tags, dto.MediaData{IsMedia: false, Media: nil})
		if err != nil {
			fmt.Println("Error summarizing X post: ", err)
			return models.Post{}, err
		}
	} else if platform == "reddit" {
		// scrapedPost, err = scraper.ScrapeRedditPost(userPost, "socks5://10.101.116.69:1088")
		scrapedPost, err = scraper.ScrapeRedditPost(userPost, "")
		if err != nil {
			fmt.Println("Error scraping Reddit post: ", err)
			return models.Post{}, err
		}
		author = scrapedPost.Author
		summary, err = s.SummarizePost(ctx, scrapedPost.Content, tags, dto.MediaData{IsMedia: false, Media: nil})
		if err != nil {
			fmt.Println("Error summarizing Reddit post: ", err)
			return models.Post{}, err
		}
	} else if platform == "instagram" {
		config := &scraper.InstagramScraperConfig{
			OutputDir: fmt.Sprintf("downloads/instagram/%d", time.Now().Unix()),
			HTTPClient: &http.Client{
				Timeout: 60 * time.Second,
				// Transport: &http.Transport{
				// 	Proxy: http.ProxyURL(&url.URL{
				// 		Scheme: "socks5",
				// 		Host:   "10.101.116.69:1088",
				// 	}),
				// },
			},
		}
		instagramScrapedPost, err := scraper.ScrapeInstagramPost(userPost, config)
		if err != nil {
			fmt.Println("Error scraping Instagram post: ", err)
			return models.Post{}, err
		}
		author = instagramScrapedPost.Author
		summary, err = s.SummarizePost(ctx, "", tags, dto.MediaData{
			IsMedia: true,
			Media:   instagramScrapedPost.Data,
		})
		if err != nil {
			fmt.Println("Error summarizing Instagram post: ", err)
			return models.Post{}, err
		}
	} else if platform == "others" {
		summary.Tags = pq.StringArray(tags)
	} else if platform == "notes" {
		summary, err = s.SummarizePost(ctx, userPost, tags, dto.MediaData{IsMedia: false, Media: nil})

		if err != nil {
			fmt.Println("Error summarizing notes: ", err)
			return models.Post{}, err
		}
	} else {
		return models.Post{}, fmt.Errorf("invalid platform")
	}

	post := models.Post{
		UserId:      ctx.GetInt64("user_id"),
		Data:        userPost,
		Author:      author,
		Topic:       summary.Topic,
		Platform:    platform,
		Category:    summary.Category,
		Tags:        summary.Tags,
		Description: summary.Description,
	}
	return post, nil
}

func (s *PostService) UpdateAuthorTagsCategories(ctx *gin.Context, post models.Post) error {
	userId := ctx.GetInt64("user_id")
	exists, err := s.postRepo.CheckUserAuthorExists(userId, post.Author, post.Platform)
	if err != nil {
		fmt.Println("Error checking user author exists: ", err)
		return err
	}
	if !exists {
		err = s.postRepo.AddUserAuthor(userId, post.Author, post.Platform)
		if err != nil {
			fmt.Println("Error adding user author: ", err)
			return err
		}
	}

	err = s.postRepo.UpdateUserTags(userId, post.Platform, post.Tags)
	if err != nil {
		fmt.Println("Error updating user tags: ", err)
		return err
	}

	err = s.postRepo.UpdateUserCategory(userId, post.Platform, post.Category)
	if err != nil {
		fmt.Println("Error updating user category: ", err)
		return err
	}
	return nil
}

func (s *PostService) CreatePost(ctx *gin.Context) {
	var request dto.CreatePostRequest
	err := ctx.ShouldBindBodyWithJSON(&request)
	if err != nil {
		fmt.Println("Error binding request body: ", err)
		utilities.Response(ctx, 400, false, nil, "Invalid request body")
		return
	}
	userPost := ""
	if request.IsUrl {
		request.Url, err = url.PathUnescape(request.Url)
		if err != nil {
			fmt.Println("Error unescaping URL: ", err)
			utilities.Response(ctx, 400, false, nil, "Invalid url")
			return
		}
		request.Url = strings.ReplaceAll(request.Url, "\\", "")
		validator := validator.New()
		if err := validator.Struct(request); err != nil {
			utilities.Response(ctx, 400, false, nil, "Invalid request body")
			return
		}
		userPost = request.Url
	} else if strings.TrimSpace(request.Notes) == "" {
		utilities.Response(ctx, 400, false, nil, "Notes are required")
		return
	} else if len(strings.TrimSpace(request.Notes)) > 3500 {
		utilities.Response(ctx, 400, false, nil, "maximum notes length is 3500 characters")
		return
	} else {
		userPost = request.Notes
	}

	platform, err := s.ExtractPostPlatform(userPost, request.IsUrl)
	if err != nil {
		fmt.Println("Error extracting post platform: ", err)
		utilities.Response(ctx, 400, false, nil, "Invalid url")
		return
	}

	post, err := s.ExtractPostDetails(ctx, userPost, platform, request.Tags)
	if err != nil {
		fmt.Println("Error extracting post details: ", err)
		utilities.Response(ctx, 400, false, nil, "Failed to extract post details")
		return
	}

	err = s.UpdateAuthorTagsCategories(ctx, post)
	if err != nil {
		fmt.Println("Error updating author tags categories: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to update author tags categories")
		return
	}

	err = s.postRepo.CreatePost(&post)
	if err != nil {
		fmt.Println("Error creating post: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to create post")
		return
	}

	postLink := fmt.Sprintf("https://lynkbin.vercel.app/dashboard?platform=%s", platform)
	response := models.CreatePostResponse{
		Post:     post,
		PostLink: postLink,
	}

	utilities.Response(ctx, 201, true, response, "Post created successfully")
}

func (s *PostService) GetPosts(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")
	var request dto.GetPostsRequest
	err := ctx.ShouldBindQuery(&request)
	if err != nil {
		fmt.Println("Error binding query parameters: ", err)
		utilities.Response(ctx, 400, false, nil, "Invalid request body")
		return
	}

	posts, err := s.postRepo.GetPosts(userId, request.Platform, request.Tags, request.Authors, request.Categories)
	if err != nil {
		fmt.Println("Error getting posts: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get posts")
		return
	}

	utilities.Response(ctx, 200, true, posts, "Posts fetched successfully")
}

func (s *PostService) GetUserAuthors(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")
	platform := ctx.Query("platform")
	if platform == "" {
		utilities.Response(ctx, 400, false, nil, "Platform is required")
		return
	}
	authors, err := s.postRepo.GetUserAuthors(userId, platform)
	if err != nil {
		fmt.Println("Error getting user authors: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get user authors")
		return
	}
	utilities.Response(ctx, 200, true, authors, "User authors fetched successfully")
}

func (s *PostService) GetUserCategories(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")
	platform := ctx.Query("platform")
	if platform == "" {
		utilities.Response(ctx, 400, false, nil, "Platform is required")
		return
	}
	categories, err := s.postRepo.GetUserCategories(userId, platform)
	if err != nil {
		fmt.Println("Error getting user categories: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get user categories")
		return
	}
	utilities.Response(ctx, 200, true, categories, "User categories fetched successfully")
}

func (s *PostService) GetUserTags(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")
	platform := ctx.Query("platform")
	if platform == "" {
		utilities.Response(ctx, 400, false, nil, "Platform is required")
		return
	}
	tags, err := s.postRepo.GetUserTags(userId, platform)
	if err != nil {
		fmt.Println("Error getting user tags: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get user tags")
		return
	}
	utilities.Response(ctx, 200, true, tags, "User tags fetched successfully")
}

func (s *PostService) GetAllUserPostsTagsAndCategoriesCount(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")
	totalPostsCount, err := s.postRepo.GetAllUserPostsCount(userId)
	if err != nil {
		fmt.Println("Error getting all user posts count: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get all user posts count")
		return
	}
	totalTagsCount, err := s.postRepo.GetAllTagsCount(userId)
	if err != nil {
		fmt.Println("Error getting all tags count: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get all tags count")
		return
	}
	totalCategoriesCount, err := s.postRepo.GetAllCategoriesCount(userId)
	if err != nil {
		fmt.Println("Error getting all categories count: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get all categories count")
		return
	}
	response := dto.GetAllTagsAndCategoriesCountResponse{
		TotalPostsCount:      totalPostsCount,
		TotalTagsCount:       totalTagsCount,
		TotalCategoriesCount: totalCategoriesCount,
	}
	utilities.Response(ctx, 200, true, response, "All counts fetched successfully")
}

func (s *PostService) DeletePost(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")

	// Get post ID from URL parameter
	postIdStr := ctx.Param("id")
	if postIdStr == "" {
		fmt.Println("Error: post ID is required")
		utilities.Response(ctx, 400, false, nil, "Post ID is required")
		return
	}

	// Convert string to int64
	postId, err := strconv.ParseInt(postIdStr, 10, 64)
	if err != nil {
		fmt.Println("Error parsing post ID: ", err)
		utilities.Response(ctx, 400, false, nil, "Invalid post ID")
		return
	}

	// Delete the post
	err = s.postRepo.DeletePost(userId, postId)
	if err != nil {
		fmt.Println("Error deleting post: ", err)
		if err.Error() == "post not found or you don't have permission to delete it" {
			utilities.Response(ctx, 404, false, nil, "Post not found or you don't have permission to delete it")
			return
		}
		utilities.Response(ctx, 500, false, nil, "Failed to delete post")
		return
	}

	utilities.Response(ctx, 200, true, nil, "Post deleted successfully")
}

func (s *PostService) GetRecentPosts(ctx *gin.Context) {
	userId := ctx.GetInt64("user_id")
	posts, err := s.postRepo.GetRecentPosts(userId)
	if err != nil {
		fmt.Println("Error getting recent posts: ", err)
		utilities.Response(ctx, 500, false, nil, "Failed to get recent posts")
		return
	}
	utilities.Response(ctx, 200, true, posts, "Recent posts fetched successfully")
}
