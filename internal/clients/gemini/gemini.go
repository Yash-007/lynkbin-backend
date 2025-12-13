package gemini

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client *genai.Client
	model  string
}

func NewGeminiClient(model string) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: "AIzaSyCfaXQtJEm86EO90--ssZwh5motDQ-Cpm0",
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
