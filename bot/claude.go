package bot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type extractedRecipe struct {
	Name        string  `json:"name"`
	Style       string  `json:"style"`
	OG          float64 `json:"og"`
	FG          float64 `json:"fg"`
	Ingredients string  `json:"ingredients"`
	Notes       string  `json:"notes"`
}

const recipePrompt = `You are a homebrewing assistant. Extract the brew recipe from the content and return ONLY a JSON object — no markdown, no explanation.

Fields:
{
  "name": "brew name or empty string",
  "style": "beer style e.g. IPA, Stout, or empty string",
  "og": original gravity float e.g. 1.065 or 0 if unknown,
  "fg": final gravity float e.g. 1.012 or 0 if not yet taken,
  "ingredients": "all ingredients as a readable list",
  "notes": "process notes, water chemistry, any other brewing details"
}`

func (b *Bot) parseRecipeFromImage(imageURL, mediaType string) (*extractedRecipe, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("downloading image: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading image: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	body := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1024,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": mediaType,
							"data":       encoded,
						},
					},
					{"type": "text", "text": recipePrompt},
				},
			},
		},
	}
	return b.callClaude(body)
}

func (b *Bot) parseRecipeFromText(text string) (*extractedRecipe, error) {
	body := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1024,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": recipePrompt + "\n\n---\n\n" + text,
			},
		},
	}
	return b.callClaude(body)
}

func (b *Bot) callClaude(body map[string]any) (*extractedRecipe, error) {
	payload, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	req.Header.Set("x-api-key", b.cfg.AnthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("claude error: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response from claude")
	}

	return parseRecipeJSON(result.Content[0].Text)
}

func parseRecipeJSON(raw string) (*extractedRecipe, error) {
	// Strip accidental markdown fences
	clean := raw
	for _, fence := range []string{"```json\n", "```json", "```\n", "```"} {
		if len(clean) >= len(fence) && clean[:len(fence)] == fence {
			clean = clean[len(fence):]
		}
		if len(clean) >= len(fence) && clean[len(clean)-len(fence):] == fence {
			clean = clean[:len(clean)-len(fence)]
		}
	}

	var r extractedRecipe
	if err := json.Unmarshal([]byte(clean), &r); err != nil {
		return nil, fmt.Errorf("parsing claude response: %w\nraw: %s", err, raw)
	}
	return &r, nil
}
