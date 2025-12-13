package utilities

import (
	"fmt"
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

// CleanJSONResponse removes markdown code blocks and extra formatting from LLM responses
func CleanJSONResponse(response string) string {
	// Trim whitespace
	cleaned := strings.TrimSpace(response)

	// Remove markdown code blocks like ```json\n{...}\n```
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")

	// Trim whitespace again after removing code blocks
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}
