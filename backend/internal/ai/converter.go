package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"remotehunter/internal/ai/prompts"
	"remotehunter/internal/models"
	"strings"
)

// ConvertResumeToTemplate takes raw text and parses it into StructuredResume via AI and builds ATS HTML template
func (c *Client) ConvertResumeToTemplate(rawText string, modelOverride string, fitSinglePage bool) (*models.StructuredResume, string, error) {
	if strings.TrimSpace(rawText) == "" {
		return nil, "", fmt.Errorf("resume text is empty")
	}

	if modelOverride == "" {
		modelOverride = c.DefaultModel()
	}

	fitInstruction := ""
	if fitSinglePage {
		fitInstruction = prompts.SinglePageFitConstraint
	}

	prompt := fmt.Sprintf(prompts.ConvertResumePromptTemplate, fitInstruction, truncate(rawText, 500000))

	rawResponse, err := c.generateCompletion(prompt, modelOverride, true)
	if err != nil {
		return nil, "", err
	}

	rawResponse = strings.TrimSpace(rawResponse)
	if idx := strings.Index(rawResponse, "{"); idx >= 0 {
		rawResponse = rawResponse[idx:]
	}
	if idx := strings.LastIndex(rawResponse, "}"); idx >= 0 {
		rawResponse = rawResponse[:idx+1]
	}

	var structRes models.StructuredResume
	if err := json.Unmarshal([]byte(rawResponse), &structRes); err != nil {
		log.Printf("[AI Client] ConvertResumeToTemplate JSON parse error: %v, raw: %s", err, rawResponse)
		return nil, "", err
	}

	htmlContent := BuildATSTemplateHTML(&structRes, fitSinglePage)
	return &structRes, htmlContent, nil
}
