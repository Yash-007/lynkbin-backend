package gemini

import (
	"context"
	"fmt"
	"io"
	"module/lynkbin/internal/dto"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client *genai.Client
	model  string
}

func NewGeminiClient(model string) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &GeminiClient{client: client, model: model}, nil
}

func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		fmt.Printf("failed to generate content: %v\n", err)
		return "", err
	}

	return result.Text(), nil
}

func (c *GeminiClient) GenerateContentWithMedia(ctx context.Context, prompt string, Media []dto.Media) (string, error) {
	parts := []*genai.Part{}

	for i, media := range Media {
		mediaPath := media.Path
		mimeType := detectMimeType(mediaPath)
		file, err := os.Open(mediaPath)
		if err != nil {
			return "", fmt.Errorf("failed to open media file: %w", err)
		}
		defer file.Close()
		mediaData, err := io.ReadAll(file)
		if err != nil {
			return "", fmt.Errorf("failed to read media file: %w", err)
		}
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{MIMEType: mimeType, Data: mediaData},
		})

		if i == len(Media)-1 {
			pathArray := strings.Split(mediaPath, "/")
			pathArray = pathArray[:len(pathArray)-1]
			folderPath := strings.Join(pathArray, "/")
			fmt.Printf("folder path: %s\n", folderPath)
			err = os.RemoveAll(folderPath)
			if err != nil {
				fmt.Printf("failed to remove folder: %v\n", err)
			}
		}
	}

	parts = append(parts, &genai.Part{
		Text: prompt,
	})

	contents := []*genai.Content{
		{
			Role:  "user",
			Parts: parts,
		},
	}

	result, err := c.client.Models.GenerateContent(ctx, c.model, contents, nil)
	if err != nil {
		fmt.Printf("failed to generate content with media: %v\n", err)
		return "", err
	}

	return result.Text(), nil
}

func detectMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".webm": "video/webm",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	return "application/octet-stream"
}
