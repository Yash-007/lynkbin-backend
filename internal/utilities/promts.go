package utilities

import (
	"fmt"
	"module/lynkbin/internal/dto"
	"path/filepath"
	"strings"
)

func GenerateCategorizationPrompt(content string, tags string, categories string, userTags string) string {
	existingTagsSection := ""
	if tags != "" {
		existingTagsSection = fmt.Sprintf(`
Existing Tags (PRIORITIZE REUSING THESE):
%s

CRITICAL RULES:
1. Before creating any new tag, check if an existing tag already covers the topic
2. ALWAYS prefer reusing existing tags when they are SPECIFIC and relevant to this post
3. If an existing tag is too generic (e.g., "Technology", "Business"), prefer creating a more SPECIFIC tag (e.g., "AI Tools", "SaaS Marketing")
4. Only create a new tag if none of the existing tags are relevant OR all existing tags are too generic for this specific post`, tags)
	}

	existingCategoriesSection := ""
	if categories != "" {
		existingCategoriesSection = fmt.Sprintf(`
Existing Categories (for reference only):
%s

You may use one of these if it fits, but feel free to suggest a new category if it better describes the post.`, categories)
	}

	userTagsSection := ""
	tagsInstruction := "Exactly 3 SPECIFIC tags that capture the key topics/themes in this post"

	if userTags != "" {
		userTagsSection = fmt.Sprintf(`
User-Provided Tags:
%s

IMPORTANT: You MUST include ALL user-provided tags in your final response. Generate additional SPECIFIC tags ONLY if needed to reach a total of 3 tags.
- If user provided 1 tag → Generate 2 more SPECIFIC tags (check existing tags first)
- If user provided 2 tags → Generate 1 more SPECIFIC tag (check existing tags first)
- If user provided 3 tags → Use all 3 user tags (no generation needed)
- When generating additional tags, prioritize reusing existing SPECIFIC tags over creating new ones`, userTags)
		tagsInstruction = "Exactly 3 tags total (including user-provided tags + specific generated/existing tags to complement them)"
	}

	return fmt.Sprintf(`You are analyzing content for a content curation app that helps users organize and revisit posts from social media.
%s
%s
%s

Analyze the following post content and provide:
1. **Category**: A single broad category (e.g., Technology, Business, Career, Health, Education, Entertainment, Personal Development, etc.)

2. **Topic**: The first few words of the post (approximately 5-10 words) that serve as a natural headline or opening statement

3. **Tags**: %s
   - **BALANCE CONSISTENCY + SPECIFICITY**: 
     * FIRST: Check existing tags and reuse them if they are SPECIFIC and relevant
     * If existing tags are too generic, create MORE SPECIFIC alternatives
     * NEVER use generic tags like "Technology", "Business", "Career" alone
   - Be HIGHLY SPECIFIC and contextual to what makes THIS post unique
   - Use simple, easy-to-understand terms (avoid complex vocabulary, but keep it professional and meaningful)
   - GOOD Examples:
     * ❌ "Career Development" → ✅ "Job Search", "Interview Tips", "Career Switch", "Resume Writing"
     * ❌ "Technology" → ✅ "AI Tools", "Machine Learning", "Cloud Computing", "API Development"
     * ❌ "Entrepreneurial Ventures" → ✅ "Startup Ideas", "Side Hustle", "Product Launch"
     * ❌ "Marketing" → ✅ "Content Marketing", "SEO Strategy", "Email Campaigns"
     * ❌ "Finance" → ✅ "Stock Trading", "Personal Budgeting", "Crypto Investment"

4. **Description**: A concise 1-2 sentence summary of what the post is about

Post Content:
%s

IMPORTANT: 
- BALANCE BOTH: Reuse existing tags when they are SPECIFIC and match the content perfectly
- SPECIFICITY IS MANDATORY: Tags MUST be highly specific and contextual (never generic like "Technology" or "Business")
- If existing tags are too generic, create specific alternatives that better capture the post's unique content
- CONSISTENCY MATTERS: Same posts should get same tags, so prefer reusing specific existing tags over creating synonyms
- Respond with ONLY the raw JSON object, no markdown formatting, no code blocks, no additional text.

JSON format:
{
  "category": "category_name",
  "topic": "first few words of the post",
  "tags": ["specific_tag1", "specific_tag2", "specific_tag3"],
  "description": "brief summary here"
}`, existingCategoriesSection, existingTagsSection, userTagsSection, tagsInstruction, content)
}

func GenerateMediaCategorizationPrompt(Media []dto.Media, tags string, categories string, userTags string) string {
	mediaCount := len(Media)

	mediaTypesMap := make(map[string]int)
	for _, media := range Media {
		mediaType := detectMediaTypeFromPath(media.Path)
		mediaTypesMap[mediaType]++
	}

	mediaTypesSummary := ""
	for mediaType, count := range mediaTypesMap {
		if mediaTypesSummary != "" {
			mediaTypesSummary += ", "
		}
		if count == 1 {
			mediaTypesSummary += fmt.Sprintf("1 %s", mediaType)
		} else {
			mediaTypesSummary += fmt.Sprintf("%d %ss", count, mediaType)
		}
	}

	multipleMediaNote := ""
	if mediaCount > 1 {
		multipleMediaNote = fmt.Sprintf("\n\nIMPORTANT: You will receive %d separate media files with individual context. Analyze ALL of them together and provide ONE SINGLE COMBINED response that captures the overall theme/message across all media.", mediaCount)
	}

	mediaInfoSection := fmt.Sprintf(`
MEDIA INFORMATION:
- Total Media Files: %d
- Media Types: %s

You are analyzing %d piece(s) of media. Consider ALL media together to understand the complete context.%s`, mediaCount, mediaTypesSummary, mediaCount, multipleMediaNote)

	mediaContextDetails := ""
	if len(Media) > 0 {
		for i, media := range Media {
			if media.Context != "" {
				if mediaContextDetails == "" {
					mediaContextDetails = "\n\nAdditional Context for Each Media:"
				}
				mediaContextDetails += fmt.Sprintf("\n  Media %d: %s", i+1, media.Context)
			}
		}
	}

	existingTagsSection := ""
	if tags != "" {
		existingTagsSection = fmt.Sprintf(`
Existing Tags (PRIORITIZE REUSING THESE):
%s

CRITICAL RULES:
1. Before creating any new tag, check if an existing tag already covers the topic
2. ALWAYS prefer reusing existing tags when they are SPECIFIC and relevant to this media
3. If an existing tag is too generic (e.g., "Technology", "Business"), prefer creating a more SPECIFIC tag (e.g., "AI Tools", "Fitness Workout")
4. Only create a new tag if none of the existing tags are relevant OR all existing tags are too generic for this specific media`, tags)
	}

	existingCategoriesSection := ""
	if categories != "" {
		existingCategoriesSection = fmt.Sprintf(`
Existing Categories (for reference only):
%s

You may use one of these if it fits, but feel free to suggest a new category if it better describes the media.`, categories)
	}

	userTagsSection := ""
	tagsInstruction := "Exactly 3 SPECIFIC tags that capture the key topics/themes in this media"

	if userTags != "" {
		userTagsSection = fmt.Sprintf(`
User-Provided Tags:
%s

IMPORTANT: You MUST include ALL user-provided tags in your final response. Generate additional SPECIFIC tags ONLY if needed to reach a total of 3 tags.
- If user provided 1 tag → Generate 2 more SPECIFIC tags (check existing tags first)
- If user provided 2 tags → Generate 1 more SPECIFIC tag (check existing tags first)
- If user provided 3 tags → Use all 3 user tags (no generation needed)
- When generating additional tags, prioritize reusing existing SPECIFIC tags over creating new ones`, userTags)
		tagsInstruction = "Exactly 3 tags total (including user-provided tags + specific generated/existing tags to complement them)"
	}

	mediaTypeDescription := ""
	if mediaCount == 1 {
		singleType := detectMediaTypeFromPath(Media[0].Path)
		switch singleType {
		case "image":
			mediaTypeDescription = "Analyze this image carefully. Look at the visual elements, subjects, composition, and overall theme."
		case "video", "reel":
			mediaTypeDescription = "Analyze this video/reel carefully. Consider the visual content, actions, scenes, mood, and overall theme."
		default:
			mediaTypeDescription = "Analyze this media content carefully."
		}
	} else {
		mediaTypeDescription = fmt.Sprintf("Analyze all %d pieces of media carefully as a cohesive set. Consider how they work together to convey a unified message or theme.", mediaCount)
	}

	return fmt.Sprintf(`You are analyzing visual media content for a content curation app that helps users organize and revisit posts from social media.
%s
%s
%s
%s
%s%s

%s

Analyze the media and provide:
1. **Category**: A single broad category (e.g., Technology, Business, Fitness, Travel, Food, Fashion, Entertainment, Education, Lifestyle, Comedy, etc.)

2. **Topic**: A brief 5-10 word description that captures what the media is about (e.g., "Morning workout routine demonstration", "Travel vlog in Paris", "Cooking pasta recipe tutorial")

3. **Tags**: %s
   - **BALANCE CONSISTENCY + SPECIFICITY**: 
     * FIRST: Check existing tags and reuse them if they are SPECIFIC and relevant
     * If existing tags are too generic, create MORE SPECIFIC alternatives
     * NEVER use generic tags like "Video Content", "Social Media", "Entertainment" alone
   - Be HIGHLY SPECIFIC and contextual to what makes THIS media unique
   - Focus on the MAIN SUBJECT, ACTIVITY, or THEME shown in the media
   - GOOD Examples:
     * ❌ "Fitness" → ✅ "Home Workout", "Yoga Practice", "Running Tips", "Strength Training"
     * ❌ "Food" → ✅ "Pasta Recipe", "Healthy Breakfast", "Street Food", "Baking Tutorial"
     * ❌ "Entertainment" → ✅ "Couple Comedy", "Dance Performance", "Music Cover", "Stand-up Comedy"
     * ❌ "Travel" → ✅ "Beach Vacation", "City Tour", "Adventure Hiking", "Hotel Review"
     * ❌ "Fashion" → ✅ "Outfit Ideas", "Makeup Tutorial", "Fashion Haul", "Styling Tips"

4. **Description**: A concise 1-2 sentence summary describing what is shown in the media, including key visual elements, actions, or themes

IMPORTANT: 
- ANALYZE THE VISUAL CONTENT: Describe what you actually see in the media
- FOR MULTIPLE MEDIA: If analyzing multiple media files, provide ONE SINGLE combined response that represents all media together
- BALANCE BOTH: Reuse existing tags when they are SPECIFIC and match the content perfectly
- SPECIFICITY IS MANDATORY: Tags MUST be highly specific and contextual (never generic like "Video" or "Content")
- If existing tags are too generic, create specific alternatives that better capture the media's unique content
- CONSISTENCY MATTERS: Similar media should get similar tags, so prefer reusing specific existing tags over creating synonyms
- Respond with ONLY the raw JSON object, no markdown formatting, no code blocks, no additional text.

JSON format:
{
  "category": "category_name",
  "topic": "brief description of what media shows",
  "tags": ["specific_tag1", "specific_tag2", "specific_tag3"],
  "description": "detailed summary of visual content"	
}`, existingCategoriesSection, existingTagsSection, userTagsSection, mediaInfoSection, mediaContextDetails, mediaTypeDescription, tagsInstruction)
}

func detectMediaTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv":
		return "video"
	case ".mp3", ".wav", ".aac", ".flac", ".ogg", ".m4a":
		return "audio"
	case ".pdf":
		return "document"
	default:
		return "media"
	}
}

func CleanJSONResponse(response string) string {
	cleaned := strings.TrimSpace(response)

	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")

	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}
