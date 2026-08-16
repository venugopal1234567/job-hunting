package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"remotehunter/internal/ai/prompts"
	"remotehunter/internal/models"
	"strings"
)

// ValidateResumeWithRecruiter acts as an independent Senior Technical Recruiter auditing the generated resume against original source resume text
func (c *Client) ValidateResumeWithRecruiter(originalText string, generated *models.StructuredResume, modelOverride string) (*models.RecruiterValidationResult, error) {
	if generated == nil {
		return nil, fmt.Errorf("generated resume is nil")
	}

	genJSON, err := json.MarshalIndent(generated, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal generated resume: %w", err)
	}

	prompt := fmt.Sprintf(prompts.RecruiterValidationPromptTemplate, truncate(originalText, 500000), string(genJSON))

	rawResponse, err := c.generateCompletion(prompt, modelOverride, true)
	if err != nil {
		return nil, fmt.Errorf("recruiter validation failed: %w", err)
	}

	rawResponse = strings.TrimSpace(rawResponse)
	if idx := strings.Index(rawResponse, "{"); idx >= 0 {
		rawResponse = rawResponse[idx:]
	}
	if idx := strings.LastIndex(rawResponse, "}"); idx >= 0 {
		rawResponse = rawResponse[:idx+1]
	}

	var valResult models.RecruiterValidationResult
	if err := json.Unmarshal([]byte(rawResponse), &valResult); err != nil {
		log.Printf("[AI Client] ValidateResumeWithRecruiter JSON parse error: %v. Raw: %s", err, rawResponse)
		return nil, fmt.Errorf("failed to parse recruiter validation response: %w", err)
	}

	return &valResult, nil
}
