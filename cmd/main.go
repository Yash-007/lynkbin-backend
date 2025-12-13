package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"module/lynkbin/internal/api"
	"module/lynkbin/internal/utilities"

	"github.com/gin-contrib/cors"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"golang.org/x/net/proxy"

	"github.com/gin-gonic/gin"
)

// createHTTPClientWithProxy creates an HTTP client with SOCKS5 or HTTP proxy support
func createHTTPClientWithProxy(proxyURL string) *http.Client {
	if proxyURL == "" {
		// No proxy, return default client
		return &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		fmt.Printf("Error parsing proxy URL: %v, using default client\n", err)
		return &http.Client{Timeout: 30 * time.Second}
	}

	var transport *http.Transport

	if parsedURL.Scheme == "socks5" {
		// SOCKS5 proxy
		dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, nil, proxy.Direct)
		if err != nil {
			fmt.Printf("Error creating SOCKS5 proxy: %v, using default client\n", err)
			return &http.Client{Timeout: 30 * time.Second}
		}

		transport = &http.Transport{
			Dial: dialer.Dial,
		}
	} else {
		// HTTP/HTTPS proxy
		transport = &http.Transport{
			Proxy: http.ProxyURL(parsedURL),
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func main() {
	godotenv.Load()
	fmt.Println("Hello, World!")

	// ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	// defer cancel()

	// telegramBotFatherToken := os.Getenv("TELEGRAM_BOTFATHER_TOKEN")

	// // Create custom HTTP client with proxy (optional)
	// httpClient := createHTTPClientWithProxy("socks5://10.76.175.247:1088")

	// opts := []bot.Option{
	// 	bot.WithDefaultHandler(handler),
	// 	bot.WithCheckInitTimeout(30 * time.Second),
	// 	bot.WithHTTPClient(30*time.Second, httpClient), // Add custom HTTP client with proxy
	// }
	// b, err := bot.New(telegramBotFatherToken, opts...)
	// if err != nil {
	// 	panic(err)
	// }
	// b.Start(ctx)

	server := gin.New()

	server.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"*"},
		AllowHeaders: []string{"*"},
	}))
	server.GET("/", func(ctx *gin.Context) {
		utilities.Response(ctx, 200, true, nil, "server is running fine")
	})

	container := api.NewContainer()
	if container == nil {
		fmt.Println("failed to create container")
		os.Exit(1)
	}

	api.RegisterRoutes(&server.RouterGroup, container)

	// server.GET("/", func(c *gin.Context) {
	// 	scraperConfig := &scraper.ScraperConfig{
	// 		Proxy: "socks5://10.101.116.69:1088",
	// 	}
	// 	// content, err := scraper.ScrapeLinkedInPost("https://www.linkedin.com/posts/meghdut-mandal-aa2625185_seems-like-folks-at-zomato-are-vibe-coding-activity-7394978339096248321-fUE0?utm_source=social_share_send&utm_medium=android_app&rcm=ACoAADlGEOABAWBa3ydvuP3qQOOrqcqshd81MIM&utm_campaign=share_via", scraperConfig)
	// 	// content, err := scraper.ScrapeLinkedInPost("https://www.linkedin.com/posts/shristi-shreya-singh-51104115b_we-have-been-so-hard-wired-to-keep-working-activity-7395839568731828226-HY7E?utm_source=social_share_send&utm_medium=android_app&rcm=ACoAADlGEOABAWBa3ydvuP3qQOOrqcqshd81MIM&utm_campaign=share_via", scraperConfig)
	// 	// content, err := scraper.ScrapeXPost("https://x.com/swarnimodi/status/1990391924461838525?s=20", scraperConfig)
	// 	// content, err := scraper.ScrapeXPost("https://x.com/piyush_trades/status/1990414283868721266?s=20", scraperConfig)
	// 	content, err := scraper.ScrapeXPost("https://x.com/yashx_404/status/1976956380402925787?s=20", scraperConfig)
	// 	if err != nil {
	// 		c.String(500, "Error: %s", err.Error())
	// 		return
	// 	}
	// 	c.JSON(200, gin.H{"content": content})
	// })

	server.Run(":8080")
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	fmt.Printf("message: %s\n", update.Message.Text)
	fmt.Printf("message Id: %d\n", update.Message.Chat.ID)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
