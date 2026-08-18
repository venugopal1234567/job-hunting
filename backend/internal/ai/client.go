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

// Client is the AI provider client for OpenAI-compatible endpoints (NVIDIA API)
type Client struct {
	nvidiaAPIKey  string
	nvidiaBaseURL string
	nvidiaModel   string
	httpClient    *http.Client
}

// NewClient creates a new AI client with NVIDIA configuration
func NewClient(nvidiaAPIKey, nvidiaBaseURL, nvidiaModel string) *Client {
	if nvidiaBaseURL == "" {
		nvidiaBaseURL = "https://integrate.api.nvidia.com/v1"
	}
	if nvidiaModel == "" {
		nvidiaModel = "z-ai/glm-5.2"
	}
	return &Client{
		nvidiaAPIKey:  nvidiaAPIKey,
		nvidiaBaseURL: nvidiaBaseURL,
		nvidiaModel:   nvidiaModel,
		httpClient: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

// DefaultModel returns the default active model name
func (c *Client) DefaultModel() string {
	if c.nvidiaModel != "" {
		return c.nvidiaModel
	}
	return "z-ai/glm-5.2"
}

// ListModels returns available NVIDIA models for display
func (c *Client) ListModels() ([]models.NvidiaModel, error) {
	defaultM := c.DefaultModel()
	return []models.NvidiaModel{
		{
			Name:   "deepseek-ai/deepseek-r1",
			Size:   0,
			Family: "nvidia",
		},
		{
			Name:   "meta/llama-3.3-70b-instruct",
			Size:   0,
			Family: "nvidia",
		},
		{
			Name:   "qwen/qwen2.5-72b-instruct",
			Size:   0,
			Family: "nvidia",
		},
		{
			Name:   defaultM,
			Size:   0,
			Family: "nvidia",
		},
		{
			Name:   "nvidia/nemotron-3.5-lightning-30b-a3b",
			Size:   0,
			Family: "nvidia",
		},
	}, nil
}

// resolveModel returns the override if non-empty, else default model
func (c *Client) resolveModel(override string) string {
	if override != "" {
		return override
	}
	return c.DefaultModel()
}

// generateCompletion routes requests to OpenAI/NVIDIA API endpoint with 429 retry backoff & fallback models
func (c *Client) generateCompletion(prompt string, modelOverride string, jsonFormat bool) (string, error) {
	primaryModel := c.resolveModel(modelOverride)
	modelsToTry := []string{primaryModel}

	// Fallback models if primary model hits rate limits or timeouts
	for _, m := range []string{"meta/llama-3.3-70b-instruct", "deepseek-ai/deepseek-r1", "qwen/qwen2.5-72b-instruct", "nvidia/nemotron-3.5-lightning-30b-a3b"} {
		if m != primaryModel {
			modelsToTry = append(modelsToTry, m)
		}
	}

	baseURL := c.nvidiaBaseURL
	if baseURL == "" {
		baseURL = "https://integrate.api.nvidia.com/v1"
	}
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

	for _, model := range modelsToTry {
		isRateLimited := false
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
				return "", fmt.Errorf("marshal openai request: %w", err)
			}

			log.Printf("[AI Client] Calling NVIDIA API: %s with model: %s", endpoint, model)

			httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
			if err != nil {
				return "", fmt.Errorf("create openai http request: %w", err)
			}
			httpReq.Header.Set("Content-Type", "application/json")
			if c.nvidiaAPIKey != "" {
				httpReq.Header.Set("Authorization", "Bearer "+c.nvidiaAPIKey)
			}

			resp, err := c.httpClient.Do(httpReq)
			if err != nil {
				lastErr = fmt.Errorf("nvidia api request failed: %w", err)
				continue
			}

			if resp.StatusCode == 429 {
				resp.Body.Close()
				lastErr = fmt.Errorf("NVIDIA API rate limit exceeded (429). Please wait a moment before retrying.")
				isRateLimited = true
				continue
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("[AI Client] NVIDIA API Error Status %d (endpoint %s, model %s): %s", resp.StatusCode, endpoint, model, string(bodyBytes))
				lastErr = fmt.Errorf("nvidia api returned status %d for model '%s' at endpoint '%s': %s", resp.StatusCode, model, endpoint, string(bodyBytes))
				break // try next fallback model
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
				lastErr = fmt.Errorf("decode nvidia response: %w", err)
				break
			}
			resp.Body.Close()

			if len(openAIResp.Choices) == 0 {
				lastErr = fmt.Errorf("empty response choices from nvidia api")
				break
			}

			return openAIResp.Choices[0].Message.Content, nil
		}

		if isRateLimited {
			// Fail fast and do not attempt other models, since the API key itself is rate-limited.
			return "", lastErr
		}
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("all ai model attempts failed")
}
