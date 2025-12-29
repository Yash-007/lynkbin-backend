package scraper

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"module/lynkbin/internal/dto"

	"github.com/PuerkitoBio/goquery"
	"github.com/playwright-community/playwright-go"
)

type InstagramScrapedPost struct {
	Author string      `json:"author"`
	Data   []dto.Media `json:"data"`
}

type ScrapedPost struct {
	Author  string `json:"author"`
	Content string `json:"content"`
	Topic   string `json:"topic"`
}

func extractAuthorFromInstagramURL(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	// This finds the 'posts/author' part
	parts := strings.Split(u.Path, "/posts/")
	if len(parts) < 2 {
		return ""
	}
	remainder := parts[1]
	authorWithPostId := remainder

	authorWithId := ""
	// The author's name is up to the first '_'
	if i := strings.Index(remainder, "_"); i != -1 {
		authorWithId = authorWithPostId[:i]
	}

	authorWithIdSlice := strings.Split(authorWithId, "-")

	if len(authorWithIdSlice) == 1 {
		return authorWithIdSlice[0]
	}

	author := strings.Join(authorWithIdSlice[:len(authorWithIdSlice)-1], " ")

	return author
}

func extractAuthorFromXPage(page playwright.Page, pageTitle string) string {
	// Method 1: Try to get from DOM selectors (most reliable)
	authorSelectors := []string{
		"article[data-testid='tweet'] div[data-testid='User-Name'] a[role='link'] span",
		"article[data-testid='tweet'] a[role='link'][href*='/'] span",
		"div[data-testid='User-Name'] a span",
		"a[role='link'] span[class*='css-1jxf684']",
	}

	for _, selector := range authorSelectors {
		authorLocator := page.Locator(selector).First()
		count, _ := authorLocator.Count()
		if count > 0 {
			authorText, err := authorLocator.TextContent()
			if err == nil && authorText != "" {
				authorText = strings.TrimSpace(authorText)
				authorText = strings.TrimPrefix(authorText, "@")
				if authorText != "" {
					fmt.Printf("Extracted author from DOM: %s\n", authorText)
					return authorText
				}
			}
		}
	}

	// Method 2: Extract from page title (fallback)
	// Expected format: "Author Name on X: tweet content"
	if strings.Contains(pageTitle, " on X") {
		parts := strings.Split(pageTitle, " on X")
		if len(parts) > 0 {
			author := strings.TrimSpace(parts[0])
			// Remove quotes if present
			author = strings.Trim(author, "\"'")
			if author != "" {
				fmt.Printf("Extracted author from title: %s\n", author)
				return author
			}
		}
	}

	// Method 3: Try meta tags
	metaAuthor, _ := page.Locator("meta[name='twitter:creator']").GetAttribute("content")
	if metaAuthor != "" {
		metaAuthor = strings.TrimPrefix(metaAuthor, "@")
		fmt.Printf("Extracted author from meta: %s\n", metaAuthor)
		return metaAuthor
	}

	// Method 4: Extract from URL
	currentURL := page.URL()
	if strings.Contains(currentURL, "x.com/") || strings.Contains(currentURL, "twitter.com/") {
		u, err := url.Parse(currentURL)
		if err == nil {
			pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" && pathParts[0] != "status" {
				fmt.Printf("Extracted author from URL: %s\n", pathParts[0])
				return pathParts[0]
			}
		}
	}

	fmt.Println("Warning: Could not extract author, using 'Unknown'")
	return "Unknown"
}

func ScrapeLinkedInPost(url string, proxy string) (ScrapedPost, error) {
	pw, err := playwright.Run()
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not launch playwright: %w", err)
	}

	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	}

	// Add proxy if provided
	if proxy != "" {
		launchOpts.Proxy = &playwright.Proxy{
			Server: proxy,
		}
	}

	browser, err := pw.Chromium.Launch(launchOpts)
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not launch browser: %w", err)
	}

	page, err := browser.NewPage()
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not create page: %w", err)
	}

	page.SetDefaultTimeout(15_000)

	page.Route("**/*", func(route playwright.Route) {
		req := route.Request()
		u := req.URL()

		if strings.Contains(u, "authwall") || strings.Contains(u, "li-auth-wall") {
			route.Abort("blockedbyclient")
			return
		}

		route.Continue()
	})

	page.SetExtraHTTPHeaders(map[string]string{
		"Referer":    "https://www.google.com/",
		"User-Agent": "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
	})

	if _, err := page.Goto(url); err != nil {
		return ScrapedPost{}, fmt.Errorf("could not navigate: %w", err)
	}

	page.Evaluate(`() => {
		document.querySelectorAll('.sign-in-modal, .modal-wormhole, .authwall, .backdrop')
			.forEach(el => el.remove());
	}`)

	fmt.Println("Page loaded")
	// fmt.Println(page.Content())
	outputPath := "internal/scraper/content_output.txt"
	contentStr, _ := page.Content()
	if err := os.WriteFile(outputPath, []byte(contentStr), 0644); err != nil {
		fmt.Printf("Error writing to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("Scraped content written to %s\n", outputPath)
	}

	// page.WaitForTimeout(2000)
	// expect(locator)

	// author, _ := page.TextContent(".update-components-actor__name")
	// content, _ := page.TextContent(".feed-shared-update-v2__description-wrapper, .feed-shared-text")

	description, err := page.Locator(`meta[name="description"]`).GetAttribute("content")
	fmt.Println("Description: ", description)
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("failed to extract meta description: %w", err)
	}

	titleContent, err := page.Locator("meta[property='og:title']").GetAttribute("content")
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("failed to extract title content: %w", err)
	}
	author := strings.Split(titleContent, " | ")[1]
	author = strings.TrimSpace(author)

	fmt.Printf("author is %s", author)
	post := ScrapedPost{
		Author:  clean(author),
		Content: clean(description),
	}

	browser.Close()
	pw.Stop()

	return post, nil
}

func ScrapeXPost(url string, proxy string) (ScrapedPost, error) {
	fmt.Println("Starting scraper...")

	pw, err := playwright.Run()
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not launch playwright: %w", err)
	}
	defer pw.Stop()

	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
		},
	}

	if proxy != "" {
		launchOpts.Proxy = &playwright.Proxy{
			Server: proxy,
		}
	}

	fmt.Println("Launching browser...")
	browser, err := pw.Chromium.Launch(launchOpts)
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	fmt.Println("Creating context...")
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
		JavaScriptEnabled: playwright.Bool(true),
		BypassCSP:         playwright.Bool(true),
	})
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not create context: %w", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not create page: %w", err)
	}

	// Stealth script BEFORE navigation
	page.AddInitScript(playwright.Script{
		Content: playwright.String(`
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
			});
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en'],
			});
			window.chrome = { runtime: {} };
		`),
	})

	fmt.Printf("Navigating to: %s\n", url)

	// Try navigation with better error handling
	response, err := page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(60000),
		WaitUntil: playwright.WaitUntilStateLoad, // Try just 'load' first
	})

	if err != nil {
		fmt.Printf("Navigation error: %v\n", err)

		content, _ := page.Content()
		os.WriteFile("error_content.html", []byte(content), 0644)

		return ScrapedPost{}, fmt.Errorf("could not navigate: %w", err)
	}

	if response != nil {
		fmt.Printf("Response status: %d\n", response.Status())
		fmt.Printf("Response URL: %s\n", response.URL())
	} else {
		fmt.Println("WARNING: No response object returned")
	}

	fmt.Println("Page loaded, waiting...")
	page.WaitForTimeout(5000)

	// Get page title
	title, _ := page.Title()
	fmt.Printf("Page title: %s\n", title)

	// Get current URL (check for redirects)
	currentURL := page.URL()
	fmt.Printf("Current URL: %s\n", currentURL)

	// Get page content
	content, err := page.Content()
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not get content: %w", err)
	}

	fmt.Printf("Content length: %d bytes\n", len(content))

	// Save for inspection
	os.WriteFile("debug_output1.html", []byte(content), 0644)

	// Check for common blocking patterns
	if strings.Contains(content, "JavaScript is not available") {
		fmt.Println("javaScript blocked by Twitter")
	}
	if strings.Contains(content, "Something went wrong") {
		fmt.Println("twitter error page")
	}
	if strings.Contains(content, "Retry") || strings.Contains(content, "Try again") {
		fmt.Println("twitter rate limit or error")
	}

	// Try to find tweet content
	tweetExists, _ := page.Locator("article[data-testid='tweet']").Count()
	fmt.Printf("Found %d tweets\n", tweetExists)

	if tweetExists == 0 {
		// Try alternative selectors
		fmt.Println("Trying alternative selectors...")

		// Check what's actually on the page
		bodyText, _ := page.Locator("body").TextContent()
		fmt.Printf("Body text preview: %s\n", bodyText[:min(200, len(bodyText))])
	}

	tweetText, _ := page.Locator("article[data-testid='tweet'] div[data-testid='tweetText']").First().TextContent()

	author := extractAuthorFromXPage(page, title)

	post := ScrapedPost{
		Content: strings.TrimSpace(tweetText),
		Author:  author,
	}

	fmt.Printf("Author of tweet is %s: ", author)

	if tweetExists > 0 {
		fmt.Println("Scraping completed successfully!")
	}
	return post, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ScrapeRedditPost(url string, proxy string) (ScrapedPost, error) {
	fmt.Println("Starting Reddit scraper...")

	pw, err := playwright.Run()
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not launch playwright: %w", err)
	}
	defer pw.Stop()

	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
		},
	}

	if proxy != "" {
		launchOpts.Proxy = &playwright.Proxy{
			Server: proxy,
		}
	}

	fmt.Println("Launching browser...")
	browser, err := pw.Chromium.Launch(launchOpts)
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
	})
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not create context: %w", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return ScrapedPost{}, fmt.Errorf("could not create page: %w", err)
	}

	fmt.Printf("Navigating to: %s\n", url)

	response, err := page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30000),
		WaitUntil: playwright.WaitUntilStateLoad,
	})

	if err != nil {
		fmt.Printf("Navigation error: %v\n", err)
		return ScrapedPost{}, fmt.Errorf("could not navigate: %w", err)
	}

	if response != nil {
		fmt.Printf("Response status: %d\n", response.Status())
	}

	fmt.Println("Page loaded, extracting content...")
	page.WaitForTimeout(2000)

	// Try to extract post title
	postTitle := ""
	titleSelector := "h1, [slot='title'], shreddit-post h1, [data-testid='post-title']"
	titleLocator := page.Locator(titleSelector).First()
	titleCount, _ := titleLocator.Count()
	if titleCount > 0 {
		postTitle, _ = titleLocator.TextContent()
	}

	// Try to extract post content/selftext
	postContent := ""
	contentSelectors := []string{
		"div[slot='text-body']",
		"div[data-testid='post-content']",
		"div.md",
		"div[data-click-id='text']",
		"shreddit-post div.md",
	}

	for _, selector := range contentSelectors {
		contentLocator := page.Locator(selector).First()
		count, _ := contentLocator.Count()
		if count > 0 {
			postContent, _ = contentLocator.TextContent()
			if postContent != "" {
				break
			}
		}
	}

	// Combine title and content
	fullContent := strings.TrimSpace(postTitle)
	if postContent != "" {
		fullContent = fullContent + "\n\n" + strings.TrimSpace(postContent)
	}

	// Extract author
	author := ""
	authorSelectors := []string{
		"[slot='authorName']",
		"[data-testid='author-name']",
		"a[data-click-id='user']",
		"shreddit-post [slot='authorName']",
	}

	for _, selector := range authorSelectors {
		authorLocator := page.Locator(selector).First()
		count, _ := authorLocator.Count()
		if count > 0 {
			author, _ = authorLocator.TextContent()
			if author != "" {
				author = strings.TrimPrefix(author, "u/")
				author = strings.TrimSpace(author)
				break
			}
		}
	}

	// Fallback: try to get from meta tags
	if fullContent == "" {
		ogDescription, _ := page.Locator("meta[property='og:description']").GetAttribute("content")
		ogTitle, _ := page.Locator("meta[property='og:title']").GetAttribute("content")

		if ogTitle != "" {
			fullContent = ogTitle
			if ogDescription != "" {
				fullContent = fullContent + "\n\n" + ogDescription
			}
		}
	}

	if author == "" {
		// Try to extract from URL or meta
		pageTitle, _ := page.Title()
		if strings.Contains(pageTitle, "by u/") {
			parts := strings.Split(pageTitle, "by u/")
			if len(parts) > 1 {
				author = strings.Split(parts[1], " ")[0]
			}
		}
	}

	fmt.Printf("Extracted - Author: %s, Content length: %d\n", author, len(fullContent))

	if fullContent == "" {
		return ScrapedPost{}, fmt.Errorf("could not extract post content")
	}

	post := ScrapedPost{
		Content: clean(fullContent),
		Author:  clean(author),
	}

	fmt.Println("Reddit scraping completed successfully!")
	return post, nil
}

func clean(s string) string {
	if s == "" {
		return ""
	}
	return string([]byte(s))
}

// Instagram Scraper Implementation

const (
	instagramUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// InstagramScraperConfig holds configuration for Instagram scraping
type InstagramScraperConfig struct {
	Proxy      string
	OutputDir  string
	HTTPClient *http.Client
}

// ScrapeInstagramPost scrapes an Instagram post and downloads the media
func ScrapeInstagramPost(postURL string, config *InstagramScraperConfig) (InstagramScrapedPost, error) {
	fmt.Println("Starting Instagram scraper...")

	if config == nil {
		config = &InstagramScraperConfig{}
	}

	if config.OutputDir == "" {
		config.OutputDir = fmt.Sprintf("downloads/instagram/%d", time.Now().Unix())
	}

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Validate URL
	if err := validateInstagramURL(postURL); err != nil {
		return InstagramScrapedPost{}, err
	}

	fmt.Printf("Fetching from: %s\n", postURL)

	// Fetch HTML
	html, err := fetchInstagramHTML(config.HTTPClient, postURL)
	if err != nil {
		return InstagramScrapedPost{}, err
	}

	fmt.Println("Successfully fetched page content")

	// Detect media type and extract accordingly
	mediaType, err := detectInstagramMediaType(html)
	if err != nil {
		return InstagramScrapedPost{}, err
	}

	fmt.Printf("Detected media type: %s\n", mediaType)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return InstagramScrapedPost{}, fmt.Errorf("failed to create output directory: %w", err)
	}

	var scrapedPost InstagramScrapedPost

	switch mediaType {
	case "reel", "video":
		scrapedPost, err = scrapeInstagramVideo(html, config)
	case "image", "carousel":
		scrapedPost, err = scrapeInstagramImages(html, config)
	default:
		return InstagramScrapedPost{}, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	if err != nil {
		return InstagramScrapedPost{}, err
	}

	fmt.Printf("Successfully scraped Instagram content\n")
	return scrapedPost, nil
}

// detectInstagramMediaType detects whether the Instagram content is a video/reel or image/carousel
func detectInstagramMediaType(htmlStr string) (string, error) {
	productTypePattern := regexp.MustCompile(`"product_type":"([^"]+)"`)
	productTypeMatches := productTypePattern.FindStringSubmatch(htmlStr)

	if len(productTypeMatches) > 1 {
		productType := productTypeMatches[1]
		switch productType {
		case "clips":
			return "reel", nil
		case "carousel_container":
			return "carousel", nil
		case "feed":
			return "image", nil
		default:
			// Check if video_versions exists
			if strings.Contains(htmlStr, `"video_versions":[{`) {
				return "video", nil
			}
			return "image", nil
		}
	}

	// Fallback: check for video or image indicators
	if strings.Contains(htmlStr, `"video_versions":[{`) {
		return "video", nil
	}

	if strings.Contains(htmlStr, `"image_versions2"`) {
		return "image", nil
	}

	return "", fmt.Errorf("could not detect media type")
}

// scrapeInstagramVideo extracts and downloads a single video
func scrapeInstagramVideo(htmlStr string, config *InstagramScraperConfig) (InstagramScrapedPost, error) {
	videoURL, author, err := extractInstagramVideoData(htmlStr)
	if err != nil {
		return InstagramScrapedPost{}, err
	}

	fmt.Printf("Found video URL\n")
	fmt.Printf("Author: %s\n", author)

	// Generate unique filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("instagram_video_%d.mp4", timestamp)
	outputPath := filepath.Join(config.OutputDir, filename)

	// Download video
	fmt.Printf("Downloading video to: %s\n", outputPath)
	if err := downloadInstagramVideo(config.HTTPClient, videoURL, outputPath); err != nil {
		return InstagramScrapedPost{}, err
	}

	fmt.Printf("Successfully downloaded video\n")

	return InstagramScrapedPost{
		Author: clean(author),
		Data: []dto.Media{
			{
				Path:    outputPath,
				Context: "Reel",
			},
		},
	}, nil
}

// scrapeInstagramImages extracts and downloads images (single or carousel)
func scrapeInstagramImages(htmlStr string, config *InstagramScraperConfig) (InstagramScrapedPost, error) {
	// debugPath := "debug_instagram_images.html"
	// if err := os.WriteFile(debugPath, []byte(htmlStr), 0644); err == nil {
	// 	fmt.Printf("Saved HTML to %s\n", debugPath)
	// } else {
	// 	fmt.Printf("Error saving HTML to %s: %v\n", debugPath, err)
	// }

	author, err := extractInstagramAuthor(htmlStr)
	if err != nil {
		fmt.Printf("Warning: Could not extract author: %v\n", err)
		author = "Unknown"
	}

	fmt.Printf("Author: %s\n", author)

	// Check if it's a carousel
	carouselPattern := regexp.MustCompile(`"carousel_media":\[(.*?)\],"location":`)
	carouselMatches := carouselPattern.FindStringSubmatch(htmlStr)

	var imageURLs []string

	fmt.Println("Length of carousel matches: ", len(carouselMatches))
	if len(carouselMatches) > 1 {
		// It's a carousel - extract multiple images
		fmt.Println("Detected carousel with multiple images")
		imageURLs, err = extractCarouselImageURLs(htmlStr)
		if err != nil {
			return InstagramScrapedPost{}, err
		}
	} else {
		// Single image post
		fmt.Println("Detected single image post")
		imageURL, err := extractSingleImageURL(htmlStr)
		if err != nil {
			return InstagramScrapedPost{}, err
		}
		imageURLs = []string{imageURL}
	}

	fmt.Printf("Found %d image(s) to download\n", len(imageURLs))

	var data []dto.Media
	// Download all images
	timestamp := time.Now().Unix()

	for i, imageURL := range imageURLs {
		filename := fmt.Sprintf("instagram_image_%d_%d.jpg", timestamp, i+1)
		outputPath := filepath.Join(config.OutputDir, filename)

		fmt.Printf("Downloading image %d/%d to: %s\n", i+1, len(imageURLs), outputPath)
		if err := downloadInstagramImage(config.HTTPClient, imageURL, outputPath); err != nil {
			fmt.Printf("Warning: Failed to download image %d: %v\n", i+1, err)
			continue
		}

		data = append(data, dto.Media{
			Path:    outputPath,
			Context: "Image",
		})

	}

	if len(data) == 0 {
		return InstagramScrapedPost{}, fmt.Errorf("failed to download any images")
	}

	fmt.Printf("Successfully downloaded %d image(s)\n", len(data))

	// For now, return the first image path, but we'll need to update this to handle multiple paths
	return InstagramScrapedPost{
		Author: clean(author),
		Data:   data,
	}, nil
}

// extractInstagramAuthor extracts the author name from Instagram HTML
func extractInstagramAuthor(htmlStr string) (string, error) {
	// Method 1: Extract from og:title
	ogTitlePattern := regexp.MustCompile(`property="og:title" content="([^"]+) on Instagram:`)
	ogTitleMatches := ogTitlePattern.FindStringSubmatch(htmlStr)
	if len(ogTitleMatches) > 1 {
		return html.UnescapeString(ogTitleMatches[1]), nil
	}

	// Method 2: Extract from twitter:title
	twitterTitlePattern := regexp.MustCompile(`name="twitter:title" content="([^"]+) \(@"`)
	twitterTitleMatches := twitterTitlePattern.FindStringSubmatch(htmlStr)
	if len(twitterTitleMatches) > 1 {
		return html.UnescapeString(twitterTitleMatches[1]), nil
	}

	// Method 3: Extract from JSON username field
	usernamePattern := regexp.MustCompile(`"username":"([^"]+)"`)
	usernameMatches := usernamePattern.FindStringSubmatch(htmlStr)
	if len(usernameMatches) > 1 {
		return usernameMatches[1], nil
	}

	return "Unknown", fmt.Errorf("could not extract author")
}

// extractCarouselImageURLs extracts all image URLs from a carousel post
func extractCarouselImageURLs(htmlStr string) ([]string, error) {
	var imageURLs []string

	// Pattern to find all image candidates in carousel_media
	imagePattern := regexp.MustCompile(`"image_versions2":\{"candidates":\[\{"url":"([^"]+)"`)
	matches := imagePattern.FindAllStringSubmatch(htmlStr, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("could not find image URLs in carousel")
	}

	// Extract unique URLs (first occurrence of each image)
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			imgURL := strings.ReplaceAll(match[1], `\/`, "/")
			imgURL = html.UnescapeString(imgURL)

			// Only add if we haven't seen this URL before
			if !seen[imgURL] {
				imageURLs = append(imageURLs, imgURL)
				seen[imgURL] = true
			}
		}
	}

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf("no valid image URLs found in carousel")
	}

	return imageURLs, nil
}

// extractSingleImageURL extracts a single image URL from a post
func extractSingleImageURL(htmlStr string) (string, error) {
	imagePattern := regexp.MustCompile(`"image_versions2":\{"candidates":\[\{"url":"([^"]+)"`)
	imageMatches := imagePattern.FindStringSubmatch(htmlStr)

	if len(imageMatches) > 1 {
		imageURL := strings.ReplaceAll(imageMatches[1], `\/`, "/")
		return html.UnescapeString(imageURL), nil
	}

	// Try og:image as fallback
	ogImagePattern := regexp.MustCompile(`property="og:image" content="([^"]+)"`)
	ogImageMatches := ogImagePattern.FindStringSubmatch(htmlStr)
	if len(ogImageMatches) > 1 {
		return html.UnescapeString(ogImageMatches[1]), nil
	}

	return "", fmt.Errorf("could not find image URL")
}

// downloadInstagramImage downloads an image from Instagram
func downloadInstagramImage(client *http.Client, imageURL, outputPath string) error {
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", instagramUserAgent)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://www.instagram.com/")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	bytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	fmt.Printf("Downloaded %d bytes\n", bytesWritten)
	return nil
}

// validateInstagramURL validates that the URL is a valid Instagram reel or post URL
func validateInstagramURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Host != "www.instagram.com" && parsedURL.Host != "instagram.com" {
		return fmt.Errorf("not an Instagram URL")
	}

	if !strings.Contains(parsedURL.Path, "/reel/") && !strings.Contains(parsedURL.Path, "/p/") {
		return fmt.Errorf("URL must be an Instagram reel or post")
	}

	return nil
}

// fetchInstagramHTML fetches the HTML content from Instagram
func fetchInstagramHTML(client *http.Client, reelURL string) (string, error) {
	req, err := http.NewRequest("GET", reelURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", instagramUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// extractInstagramVideoData extracts video URL and author from Instagram's embedded JSON
func extractInstagramVideoData(html string) (videoURL, author string, err error) {
	// Save HTML for debugging
	// debugPath := "debug_instagram_video.html"
	// if err := os.WriteFile(debugPath, []byte(html), 0644); err == nil {
	// 	fmt.Printf("Saved HTML to %s\n", debugPath)
	// } else {
	// 	fmt.Printf("Error saving HTML to %s: %v\n", debugPath, err)
	// }

	// Method 1: Find video_versions in embedded JSON (new Instagram format)
	videoPattern := regexp.MustCompile(`"video_versions":\[{"width":\d+,"height":\d+,"url":"([^"]+)"`)
	videoMatches := videoPattern.FindStringSubmatch(html)
	if len(videoMatches) > 1 {
		videoURL = strings.ReplaceAll(videoMatches[1], `\/`, "/")
	}

	author, err = extractInstagramAuthor(html)
	if err != nil {
		author = "Unknown"
	}

	// Method 2: Try application/ld+json if regex fails
	if videoURL == "" {
		doc, parseErr := goquery.NewDocumentFromReader(strings.NewReader(html))
		if parseErr == nil {
			doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
				jsonStr := s.Text()
				var data map[string]interface{}
				if jsonErr := json.Unmarshal([]byte(jsonStr), &data); jsonErr == nil {
					if contentURL, ok := data["contentUrl"].(string); ok && videoURL == "" {
						videoURL = contentURL
					}
					if authorData, ok := data["author"].(map[string]interface{}); ok && author == "" {
						if authorName, ok := authorData["name"].(string); ok {
							author = authorName
						}
					}
				}
			})
		}
	}

	if videoURL == "" {
		return "", "", fmt.Errorf("could not find video URL in the page")
	}

	if author == "" {
		author = "Unknown"
	}

	return videoURL, author, nil
}

// downloadInstagramVideo downloads the video from the given URL
func downloadInstagramVideo(client *http.Client, videoURL, outputPath string) error {
	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("User-Agent", instagramUserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://www.instagram.com/")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code while downloading: %d", resp.StatusCode)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Download video to file
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write video to file: %w", err)
	}

	fmt.Printf("Downloaded %d bytes\n", written)
	return nil
}
