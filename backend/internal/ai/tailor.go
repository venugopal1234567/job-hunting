package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"remotehunter/internal/ai/prompts"
	"remotehunter/internal/models"
	"strings"
	"time"
)

// AnalyzeATSMatch sends a structured prompt to the AI provider
// and returns a structured ATS analysis report.
func (c *Client) AnalyzeATSMatch(job *models.Job, resume *models.Resume, modelOverride string) (*models.ATSAnalysis, error) {
	prompt := buildATSPrompt(job, resume)
	rawResponse, err := c.generateCompletion(prompt, modelOverride, true)
	if err != nil {
		return nil, err
	}
	return parseATSResponse(rawResponse, job, resume)
}

// buildATSPrompt constructs a structured JSON-requesting prompt for any job/resume
func buildATSPrompt(job *models.Job, resume *models.Resume) string {
	resumeSkills := strings.Join(resume.ExtractedSkills, ", ")

	return fmt.Sprintf(prompts.ATSMatchPromptTemplate,
		job.Title,
		job.Company,
		truncate(job.Description, 500000),
		resumeSkills,
		truncate(resume.RawText, 500000),
	)
}

// parseATSResponse parses the LLM JSON response into an ATSAnalysis struct
func parseATSResponse(raw string, job *models.Job, resume *models.Resume) (*models.ATSAnalysis, error) {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "{"); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 {
		raw = raw[:idx+1]
	}

	var parsed struct {
		ATSScore              int                  `json:"ats_score"`
		MatchedSkills         []string             `json:"matched_skills"`
		MissingSkills         []string             `json:"missing_skills"`
		ActionableSuggestions []string             `json:"actionable_suggestions"`
		GapQuestions          []models.GapQuestion `json:"gap_questions"`
	}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w. raw: %s", err, raw)
	}

	return &models.ATSAnalysis{
		JobID:    job.ID,
		ResumeID: resume.ID,
		ATSScore: parsed.ATSScore,
		MatchBreakdown: models.MatchBreakdown{
			MatchedSkills: parsed.MatchedSkills,
			MissingSkills: parsed.MissingSkills,
		},
		ActionableSuggestions: parsed.ActionableSuggestions,
		GapQuestions:          parsed.GapQuestions,
		AnalyzedAt:            time.Now(),
	}, nil
}

// ChatWithResume sends a conversational message with resume context to AI provider.
func (c *Client) ChatWithResume(req *models.ChatRequest, jobContext string, modelOverride string) (*models.ChatResponse, error) {
	prompt := buildChatPrompt(req, jobContext)
	rawResponse, err := c.generateCompletion(prompt, modelOverride, true)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	log.Printf("[AI] Raw chat response:\n%s", rawResponse)
	resp := parseChatResponse(rawResponse)
	if resp != nil {
		resp.ProposedEdits = []models.ProposedEdit{}
	}
	return resp, nil
}

// buildChatPrompt constructs the resume chat prompt
func buildChatPrompt(req *models.ChatRequest, jobContext string) string {
	jobSection := ""
	if jobContext != "" {
		jobSection = fmt.Sprintf("\n\nTARGET JOB DESCRIPTION:\n%s", truncate(jobContext, 500000))
	}

	var cleanResumeText string
	if req.ResumeStructured != nil {
		cleanResumeText = structuredToTextGo(req.ResumeStructured)
	} else {
		cleanResumeText = stripHTMLForPrompt(req.ResumeText)
	}

	return fmt.Sprintf(prompts.ChatResumePromptTemplate,
		truncate(cleanResumeText, 500000),
		jobSection,
		req.Message,
	)
}

// parseChatResponse parses the LLM JSON chat response
func parseChatResponse(raw string) *models.ChatResponse {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "{"); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 {
		raw = raw[:idx+1]
	}

	var parsed models.ChatResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		log.Printf("[AI] Failed to parse chat JSON: %v. Raw output was:\n%s", err, raw)
		return &models.ChatResponse{
			Message: "I had trouble structuring my response. Please try again.",
		}
	}
	return &parsed
}
