package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Davincible/goinsta/v3"
	"github.com/spf13/viper"
)

type openRouterRequest struct {
	Model    string              `json:"model"`
	Messages []openRouterMessage `json:"messages"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message openRouterMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (myInstabot MyInstabot) generateAISuggestion(image goinsta.Item, userInfo *goinsta.User) string {
	caption := strings.TrimSpace(image.Caption.Text)
	if caption == "" {
		caption = "no caption"
	}

	prompt := fmt.Sprintf(
		`You are an enthusiastic Instagram user browsing photos. Write ONE short, natural comment (max 15 words) for this post.

Post caption: "%s"
Username: %s
Followers: %d
Likes on post: %d

Rules:
- Be genuine, positive and expressive
- Refer to the caption content if possible
- Use 1-3 relevant emojis that fit the vibe — they make comments feel alive
- Don't be bland or robotic
- Sound like a real enthusiastic person
- Reply with ONLY the comment text, nothing else`,
		caption, userInfo.Username, userInfo.FollowerCount, image.Likes,
	)

	return generateAIComment(prompt)
}

func generateAIComment(prompt string) string {
	apiKey := viper.GetString("openrouter.api_key")
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		log.Println("No OpenRouter API key configured, falling back to random comment")
		return ""
	}

	body := openRouterRequest{
		Model: "auto",
		Messages: []openRouterMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		return ""
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/VictorMgaya/AI_Instabot")
	req.Header.Set("X-Title", "AI Instabot")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("OpenRouter API error: %v", err)
		return ""
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		return ""
	}

	var result openRouterResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("Failed to parse response: %v", err)
		return ""
	}

	if result.Error != nil {
		log.Printf("OpenRouter API error: %s", result.Error.Message)
		return ""
	}

	if len(result.Choices) == 0 {
		log.Println("OpenRouter returned no choices")
		return ""
	}

	comment := strings.TrimSpace(result.Choices[0].Message.Content)
	comment = strings.Trim(comment, `"'`)

	log.Printf("AI generated comment: %s", comment)
	return comment
}
