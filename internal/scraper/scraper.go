package scraper

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type ScrapedPost struct {
	Author  string `json:"author"`
	Content string `json:"content"`
	Topic   string `json:"topic"`
}

func extractAuthorFromLinkedInURL(link string) string {
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
		Headless: playwright.Bool(false), // CRITICAL: Keep false to see what's happening
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
	// author, _ := page.Locator("div[class='css-146c3p1 r-bcqeeo r-1ttztb7 r-qvutc0 r-37j5jr r-a023e6 r-rjixqe r-b88u0q r-1awozwy r-6koalj r-1udh08x r-3s2u2q']").TextContent()
	authorArray := strings.Split(title, "on X")
	author := strings.TrimSpace(authorArray[0])

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
