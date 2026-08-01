package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"remotehunter/internal/models"
	"strings"
	"time"
)

// Client is the Ollama API client
type Client struct {
	host       string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Ollama client
func NewClient(host, model string) *Client {
	return &Client{
		host:  host,
		model: model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// AnalyzeATSMatch sends a structured prompt to the local Ollama Gemma model
// and returns a structured ATS analysis report
func (c *Client) AnalyzeATSMatch(job *models.Job, resume *models.Resume) (*models.ATSAnalysis, error) {
	prompt := buildATSPrompt(job, resume)

	reqBody, err := json.Marshal(ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.host+"/api/generate",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		log.Printf("[AI] Ollama unavailable, using fallback analysis: %v", err)
		return fallbackAnalysis(job, resume), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AI] Ollama returned %d, using fallback", resp.StatusCode)
		return fallbackAnalysis(job, resume), nil
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	return parseATSResponse(ollamaResp.Response, job, resume)
}

// buildATSPrompt constructs a structured JSON-requesting prompt
func buildATSPrompt(job *models.Job, resume *models.Resume) string {
	resumeSkills := strings.Join(resume.ExtractedSkills, ", ")

	return fmt.Sprintf(`You are an expert ATS (Applicant Tracking System) resume analyst.

Analyze the following job description against the candidate's resume and respond ONLY with a valid JSON object matching exactly this schema:
{
  "ats_score": <integer 0-100>,
  "matched_skills": [<list of skills from resume that match the job>],
  "missing_skills": [<list of skills the job requires that the candidate lacks>],
  "actionable_suggestions": [<2-4 specific bullet points to improve the resume for this role>],
  "gap_questions": [
    {"skill": "<skill name>", "question": "<specific interview prep question about that skill gap>"}
  ]
}

JOB TITLE: %s
COMPANY: %s
JOB DESCRIPTION:
%s

CANDIDATE SKILLS: %s
CANDIDATE RESUME EXCERPT:
%s

Respond only with the JSON object, no other text.`,
		job.Title,
		job.Company,
		truncate(job.Description, 2000),
		resumeSkills,
		truncate(resume.RawText, 1500),
	)
}

// parseATSResponse parses the LLM JSON response into an ATSAnalysis struct
func parseATSResponse(raw string, job *models.Job, resume *models.Resume) (*models.ATSAnalysis, error) {
	// Extract JSON from potential markdown code blocks
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
		log.Printf("[AI] Failed to parse LLM JSON response, using fallback: %v", err)
		return fallbackAnalysis(job, resume), nil
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

// fallbackAnalysis returns a basic analysis when Ollama is unavailable
func fallbackAnalysis(job *models.Job, resume *models.Resume) *models.ATSAnalysis {
	matched := []string{}
	missing := []string{}

	jobDescLower := strings.ToLower(job.Description + " " + job.Title)
	for _, skill := range resume.ExtractedSkills {
		if strings.Contains(jobDescLower, strings.ToLower(skill)) {
			matched = append(matched, skill)
		}
	}

	// Basic score: % of resume skills that appear in job description
	score := 0
	if len(resume.ExtractedSkills) > 0 {
		score = (len(matched) * 100) / len(resume.ExtractedSkills)
	}

	return &models.ATSAnalysis{
		JobID:    job.ID,
		ResumeID: resume.ID,
		ATSScore: score,
		MatchBreakdown: models.MatchBreakdown{
			MatchedSkills: matched,
			MissingSkills: missing,
		},
		ActionableSuggestions: []string{
			"Tailor your resume summary to match the job title.",
			"Quantify your achievements with metrics where possible.",
		},
		GapQuestions: []models.GapQuestion{},
		AnalyzedAt:   time.Now(),
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
