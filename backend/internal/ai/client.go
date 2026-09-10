package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"remotehunter/internal/models"
	"strings"
	"time"
)

// Client is the AI provider client for OpenAI-compatible endpoints
type Client struct {
	apiKey  string
	baseURL string
	model   string
	httpClient *http.Client
}

// NewClient creates a new AI client
func NewClient(apiKey, baseURL, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

// DefaultModel returns the configured model name
func (c *Client) DefaultModel() string {
	return c.model
}

// ListModels returns available AI models for display
func (c *Client) ListModels() ([]models.AIModel, error) {
	defaultM := c.DefaultModel()
	return []models.AIModel{
		{Name: defaultM, Size: 0, Family: "custom"},
	}, nil
}

// resolveModel returns the override if non-empty, else default model
func (c *Client) resolveModel(override string) string {
	if override != "" {
		return override
	}
	return c.DefaultModel()
}

// generateCompletion calls the configured OpenAI-compatible endpoint with 429 retry backoff
func (c *Client) generateCompletion(prompt string, modelOverride string, jsonFormat bool) (string, error) {
	model := c.resolveModel(modelOverride)

	baseURL := c.baseURL
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	type openAIMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type openAIReq struct {
		Model          string          `json:"model"`
		Messages       []openAIMessage `json:"messages"`
		Temperature    float64         `json:"temperature"`
		MaxTokens      int             `json:"max_tokens,omitempty"`
		ResponseFormat interface{}     `json:"response_format,omitempty"`
	}

	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			sleepSec := time.Duration(attempt*4) * time.Second
			log.Printf("[AI Client] Rate limited (429). Retrying attempt %d in %v for model %s...", attempt, sleepSec, model)
			time.Sleep(sleepSec)
		}

		reqObj := openAIReq{
			Model: model,
			Messages: []openAIMessage{
				{Role: "user", Content: prompt},
			},
			Temperature: 0.2,
			MaxTokens:   8192,
		}

		reqBody, err := json.Marshal(reqObj)
		if err != nil {
			return "", fmt.Errorf("marshal request: %w", err)
		}

		log.Printf("[AI Client] Calling AI API: %s with model: %s", endpoint, model)

		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return "", fmt.Errorf("create http request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("ai api request failed: %w", err)
			continue
		}

		if resp.StatusCode == 429 {
			resp.Body.Close()
			lastErr = fmt.Errorf("API rate limit exceeded (429). Please wait a moment before retrying.")
			continue
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Printf("[AI Client] API Error Status %d (endpoint %s, model %s): %s", resp.StatusCode, endpoint, model, string(bodyBytes))
			lastErr = fmt.Errorf("ai api returned status %d for model '%s' at endpoint '%s': %s", resp.StatusCode, model, endpoint, string(bodyBytes))
			return "", lastErr
		}

		var openAIResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("decode response: %w", err)
			break
		}
		resp.Body.Close()

		if len(openAIResp.Choices) == 0 {
			lastErr = fmt.Errorf("empty response choices from ai api")
			break
		}

		return openAIResp.Choices[0].Message.Content, nil
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("all ai model attempts failed")
}
