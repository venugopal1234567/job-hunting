package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
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
			Timeout: 300 * time.Second,
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
			Name:   "nvidia/nemotron-3.5-lightning-30b-a3b",
			Size:   0,
			Family: "nvidia",
		},
		{
			Name:   defaultM,
			Size:   0,
			Family: "nvidia",
		},
		{
			Name:   "openai/gpt-oss-120b",
			Size:   0,
			Family: "nvidia",
		},
		{
			Name:   "meta/llama-3.1-70b-instruct",
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

	// Fallback models if primary model hits rate limits
	for _, m := range []string{"meta/llama-3.1-70b-instruct", "nvidia/nemotron-3.5-lightning-30b-a3b", "openai/gpt-oss-120b"} {
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
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				sleepSec := time.Duration(attempt*2) * time.Second
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
				lastErr = fmt.Errorf("nvidia api returned status 429 for model '%s'", model)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("[AI Client] NVIDIA API Error Status %d (endpoint %s, model %s): %s", resp.StatusCode, endpoint, model, string(bodyBytes))
				lastErr = fmt.Errorf("nvidia api returned status %d for model '%s' at endpoint '%s': %s", resp.StatusCode, model, endpoint, string(bodyBytes))
				break // Non-429 error, try next fallback model
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
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("all ai model attempts failed")
}

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

	// Note: Escaped % symbols in the prompt text as %% so fmt.Sprintf doesn't break
	return fmt.Sprintf(`You are an extremely strict, literal Applicant Tracking System (ATS) algorithm. You are ruthless in your evaluation.

Your task is to analyze the Candidate Resume against the Job Description and calculate a realistic ATS Match Score. 
LLMs usually inflate scores. You must NOT inflate the score. A resume from a different sub-field (e.g., Backend Engineering vs. Test Automation) should score poorly (under 70%%), even if the candidate is highly experienced.

SCORING ALGORITHM (Start at 100, apply deductions):
SCORING ALGORITHM (Start at 100, apply deductions):
1. Domain & Title Match (Deduct up to 30 points): If the candidate's recent job titles and core daily work do NOT perfectly match the target role's specific domain, DEDUCT 20-30 points immediately. 
2. Missing Mandatory Tools/Frameworks (Deduct 5 points EACH): Identify the top 5 specific tools, frameworks, or protocols in the job description. For EVERY one missing from the resume, deduct 5 points. (General concepts do not count. A generic tool does not count for a specific one).
3. Experience & Seniority (Deduct up to 20 points): Deduct points if years of experience or scope of responsibility don't align.

HARD CAP: If the candidate is transitioning between domains (e.g., Software Engineering to Network Automation) or is missing more than half of the specific technical keywords, the final score MUST be between 40 and 70.

Respond ONLY with a valid JSON object matching exactly this schema:
{
  "score_reasoning": "<1-2 sentences explaining exactly what points were deducted for missing domains or specific tools. Write this FIRST.>",
  "ats_score": <integer from 0 to 100>,
  "matched_skills": ["<skill 1>", "<skill 2>"],
  "missing_skills": ["<missing specific framework/skill 1>", "<missing specific framework/skill 2>"],
  "actionable_suggestions": ["<specific resume edit 1>", "<specific resume edit 2>"],
  "gap_questions": [
    {
      "skill": "<missing skill name>",
      "question": "<interview prep question>"
    }
  ]
}

JOB TITLE: %s
COMPANY: %s
JOB DESCRIPTION:
%s

CANDIDATE SKILLS: %s
CANDIDATE RESUME EXCERPT:
%s

OUTPUT STRICTLY JSON:`,
		job.Title,
		job.Company,
		truncate(job.Description, 500000),
		resumeSkills,
		truncate(resume.RawText, 500000),
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ChatWithResume sends a conversational message with resume context to AI provider.
// modelOverride is optional; pass "" to use the client's default model.
func (c *Client) ChatWithResume(req *models.ChatRequest, jobContext string, modelOverride string) (*models.ChatResponse, error) {
	prompt := buildChatPrompt(req, jobContext)
	rawResponse, err := c.generateCompletion(prompt, modelOverride, true)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	log.Printf("[AI] Raw chat response:\n%s", rawResponse)
	return parseChatResponse(rawResponse), nil
}

// buildChatPrompt constructs the resume chat prompt
func buildChatPrompt(req *models.ChatRequest, jobContext string) string {
	jobSection := ""
	if jobContext != "" {
		jobSection = fmt.Sprintf("\n\nTARGET JOB DESCRIPTION:\n%s", truncate(jobContext, 500000))
	}

	return fmt.Sprintf(`You are a Senior Technical Recruiter and ATS Resume Analyst. 
Your goal is to help the candidate perfectly tailor their resume for the target Job Description into an elegant, high-converting HTML/ATS resume format.

You are interacting with the candidate via a specialized chat interface that supports inline resume editing and AI-structured section formatting.

BEHAVIOR & WORKFLOW (Strict 2-Phase Decision Tree):
- PHASE 1 (Discover Gaps First): Compare the candidate's resume against the Job Description. If there are any missing skills, technologies, or concepts required by the Job Description, ask the candidate questions in the "gap_prompts" field to see if they have this experience. If "gap_prompts" is populated, keep "full_resume_replacement" empty "".
- PHASE 2 (Complete Resume Replacement & Structured Output): Once questions are answered or when tailoring, generate the complete, fully rewritten, clean, highly optimized resume matching a high ATS score (90%%+) in BOTH "full_resume_replacement" and "structured_resume".

Respond ONLY with a valid JSON object matching exactly this schema:
{
  "message": "<Your conversational response. Explain your thoughts, feedback, or what you are changing.>",
  "proposed_edits": [
    {
      "id": "edit-1",
      "original": "<exact string excerpt from original resume>",
      "replacement": "<new updated bullet or section text>",
      "reason": "<short explanation why this improves ATS match>"
    }
  ],
  "gap_prompts": [
    {
      "skill": "<missing skill name>",
      "question": "<friendly question asking if candidate has experience with this>"
    }
  ],
  "full_resume_replacement": "<Complete updated plain text resume if replacing whole text, or empty string>",
  "structured_resume": {
    "name": "<Candidate Full Name>",
    "title": "<Candidate Target Professional Title>",
    "contact_items": ["<Phone>", "<Email>", "<Location>"],
    "summary": "<Professional Summary text tailored to job requirements with key metrics>",
    "skills": [
      {
        "category": "Databases",
        "items": "PostgreSQL, DynamoDB, Redis, MongoDB, NATS, Google Pub/Sub"
      },
      {
        "category": "Frameworks & Libraries",
        "items": "Kubernetes, Docker, Helm"
      },
      {
        "category": "Programming Languages",
        "items": "Go, Python, TypeScript, SQL, Shell Scripting"
      },
      {
        "category": "Soft Skills",
        "items": "Platform Engineering, Test-Driven Development (TDD), System Design, Agile/Scrum, Remote Collaboration"
      },
      {
        "category": "Tools & Platforms",
        "items": "AWS, Azure, GCP, CI/CD Automation"
      }
    ],
    "work_experience": [
      {
        "title": "<Job Title>",
        "date": "<Dates, e.g. Jun 2023 - Present>",
        "company": "<Company Name>",
        "location": "<Location>",
        "bullets": [
          "<Punchy, metric-driven bullet starting with strong action verb>",
          "<Another high impact bullet point>"
        ],
        "tech_stack": "Go, Kubernetes, Google Pub/Sub, Redis"
      }
    ],
    "education": [
      {
        "institution": "<College/University Name>",
        "date": "<Years, e.g. 2015 - 2019>",
        "degree": "<Degree Name>"
      }
    ],
    "highlight_keywords": [
      "Go", "Kubernetes", "Google Pub/Sub", "Redis", "Microservices", "TDD", "Docker"
    ]
  }
}

CRITICAL RULES & CONSTRAINTS:
- MANDATORY ZERO WORK EXPERIENCE LOSS: You MUST preserve EVERY SINGLE work experience entry and company from the candidate's original resume. NEVER delete, omit, or strip off any company or job role (e.g., EPAM Systems Backend Developer, EPAM Systems Portability Engineer, and InTimeTec GoLang Developer MUST ALL BE INCLUDED).
- PRESERVE OFFICIAL JOB TITLES & NO HALLUCINATIONS: Keep the exact official job titles from the candidate's original resume (e.g. "Senior Software Development Engineer"). Do NOT change official job titles to match the target job title (e.g., do NOT change "Senior Software Development Engineer" to "Senior Golang Developer"). Do NOT invent fake technical details or assign tools to specific roles where they were not listed in the original source text.
- SECTION ORDERING IS MANDATORY: 1. Professional Summary, 2. Work Experiences, 3. Educations, 4. Skills.
- The "highlight_keywords" array MUST contain all high-value keywords, technologies, skills, and metrics that should be highlighted on the resume for maximum ATS/recruiter impact.
- All bullet points in work_experience MUST be strong, concise, and impact-oriented sentences tailored to the target job description.
- Ensure proper spacing after punctuation.
- Return [] for "gap_prompts" and null for "structured_resume" if not generating a resume edit.

CURRENT RESUME:
%s
%s

USER MESSAGE: %s

OUTPUT STRICTLY JSON:`,
		truncate(req.ResumeText, 500000),
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
		fitInstruction = `
CRITICAL SINGLE PAGE FIT & ZERO LOSS CONSTRAINTS:
1. YOU MUST PRESERVE EVERY SINGLE WORK EXPERIENCE ENTRY AND COMPANY FROM THE ORIGINAL RESUME. NEVER STRIP OFF, DELETE, OR OMIT ANY COMPANY OR JOB EXPERIENCE (e.g. EPAM Systems Backend Developer, EPAM Systems Portability Engineer, and InTimeTec GoLang Developer MUST ALL BE PRESERVED).
2. To fit strictly on 1 single letter page, condense each bullet point into punchy, high-impact single-line achievements. Eliminate filler words and redundant fluff, but DO NOT drop entire bullet points or whole job experiences.
3. Categorize skills into clean, standardized categories matching: "Databases", "Frameworks & Libraries", "Programming Languages", "Soft Skills", "Tools & Platforms".
4. Ensure "tech_stack" is populated for every work experience entry.`
	}

	prompt := fmt.Sprintf(`You are an expert ATS resume parser and formatter.
Extract all information from the candidate's resume and return it strictly as a JSON object matching this exact schema:

{
  "name": "<Candidate Full Name>",
  "title": "<Candidate Current / Target Job Title, e.g. Senior Software Development Engineer>",
  "contact_items": ["<Phone Number>", "<Email Address>", "<Location (City, State/Country)>"],
  "summary": "<Professional summary paragraph>",
  "work_experience": [
    {
      "title": "<Job Role / Title>",
      "date": "<Start Date - End Date>",
      "company": "<Company Name>",
      "location": "<Location>",
      "bullets": ["<Bullet point 1>", "<Bullet point 2>"],
      "tech_stack": "<Comma-separated technologies/skills used>"
    }
  ],
  "education": [
    {
      "institution": "<College / University Name | Location>",
      "date": "<Year Range, e.g., 2015 - 2019>",
      "degree": "<Degree Name>"
    }
  ],
  "skills": [
    {
      "category": "<Category Name, e.g., Databases, Frameworks & Libraries, Programming Languages, Soft Skills, Tools & Platforms>",
      "items": "<Comma-separated skills in this category>"
    }
  ]
}

CRITICAL FORMATTING CONSTRAINTS:
1. Contact items must be clean strings without pipe ('|') separators.
2. YOU MUST EXTRACT ALL WORK EXPERIENCES AND COMPANIES FROM THE RESUME. DO NOT OMIT OR STRIP ANY JOB ROLE.
3. Job Title must be present for every entry.
4. Company Name must be present for every entry.
5. Technologies / Skills Used must be populated on tech_stack for each job.
6. Skill categories must be organized cleanly (Databases, Frameworks & Libraries, Programming Languages, Soft Skills, Tools & Platforms).

Remove all citation markers like [cite: 1].%s
CANDIDATE RESUME TEXT:
%s

OUTPUT STRICTLY VALID JSON:`, fitInstruction, truncate(rawText, 500000))

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

func renderFormattedTextGo(str string) string {
	s := regexp.MustCompile(`^[•\-*▪◦\s]+`).ReplaceAllString(str, "")
	s = html.EscapeString(strings.TrimSpace(s))
	s = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(s, "<strong>$1</strong>")
	s = regexp.MustCompile(`(?i)&lt;strong&gt;(.*?)&lt;/strong&gt;`).ReplaceAllString(s, "<strong>$1</strong>")
	s = regexp.MustCompile(`(?i)&lt;b&gt;(.*?)&lt;/b&gt;`).ReplaceAllString(s, "<strong>$1</strong>")
	return s
}

func formatBulletActionVerbGo(str string) string {
	if str == "" {
		return ""
	}
	s := regexp.MustCompile(`^[•\-*▪◦\s]+`).ReplaceAllString(str, "")
	s = html.EscapeString(strings.TrimSpace(s))

	if regexp.MustCompile(`^\*\*(.*?)\*\*`).MatchString(s) {
		s = regexp.MustCompile(`^\*\*(.*?)\*\*`).ReplaceAllString(s, "<strong>$1</strong>")
		s = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(s, "<strong>$1</strong>")
		return s
	}
	if regexp.MustCompile(`(?i)^&lt;strong&gt;(.*?)&lt;/strong&gt;`).MatchString(s) {
		s = regexp.MustCompile(`(?i)&lt;strong&gt;(.*?)&lt;/strong&gt;`).ReplaceAllString(s, "<strong>$1</strong>")
		s = regexp.MustCompile(`(?i)&lt;b&gt;(.*?)&lt;/b&gt;`).ReplaceAllString(s, "<strong>$1</strong>")
		return s
	}

	s = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(s, "<strong>$1</strong>")
	s = regexp.MustCompile(`(?i)&lt;strong&gt;(.*?)&lt;/strong&gt;`).ReplaceAllString(s, "<strong>$1</strong>")

	parts := strings.SplitN(s, " ", 2)
	if len(parts) > 0 && regexp.MustCompile(`^[A-Za-z0-9\-\/]+$`).MatchString(parts[0]) && !strings.HasPrefix(parts[0], "<strong>") {
		firstWord := parts[0]
		rest := ""
		if len(parts) > 1 {
			rest = " " + parts[1]
		}
		if regexp.MustCompile(`^[A-Za-z]`).MatchString(firstWord) {
			return "<strong>" + firstWord + "</strong>" + rest
		}
	}

	return s
}

func formatJobTitleLineGo(title string) string {
	if title == "" {
		return ""
	}
	if strings.Contains(title, "|") {
		parts := strings.Split(title, "|")
		mainTitle := strings.TrimSpace(parts[0])
		restType := strings.TrimSpace(strings.Join(parts[1:], " | "))
		return "<strong>" + html.EscapeString(mainTitle) + "</strong> | " + html.EscapeString(restType)
	}
	return "<strong>" + html.EscapeString(title) + "</strong>"
}

// BuildATSTemplateHTML renders a structured resume into exact Times New Roman HTML/CSS template
func BuildATSTemplateHTML(sr *models.StructuredResume, fitSinglePage ...bool) string {
	if sr == nil {
		return ""
	}

	isSinglePage := len(fitSinglePage) > 0 && fitSinglePage[0]

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>`)
	sb.WriteString(html.EscapeString(sr.Name))
	sb.WriteString(` - Resume</title>
    <style>
        @page {
            size: letter;
            margin: `)
	if isSinglePage {
		sb.WriteString(`12px 18px;`)
	} else {
		sb.WriteString(`18px 20px;`)
	}
	sb.WriteString(`
        }
        body {
            font-family: "Times New Roman", Times, serif;
            color: #000;
            line-height: `)
	if isSinglePage {
		sb.WriteString(`1.2; font-size: 13px; padding: 10px 15px;`)
	} else {
		sb.WriteString(`1.25; font-size: 13.5px; padding: 18px 20px;`)
	}
	sb.WriteString(`
            max-width: 850px;
            margin: 0 auto;
            background-color: #fff;
        }
        
        /* Header Styling */
        header {
            text-align: center;
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`8px;`)
	} else {
		sb.WriteString(`12px;`)
	}
	sb.WriteString(`
        }
        h1 {
            font-family: "Times New Roman", Times, serif;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`24px;`)
	} else {
		sb.WriteString(`26px;`)
	}
	sb.WriteString(`
            font-weight: bold;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin: 0 0 3px 0;
            text-align: center;
        }
        .subtitle {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`14px; margin: 0 0 4px 0;`)
	} else {
		sb.WriteString(`15px; margin: 0 0 6px 0;`)
	}
	sb.WriteString(`
            text-align: center;
        }
        .contact-info {
            font-family: "Times New Roman", Times, serif;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`12px;`)
	} else {
		sb.WriteString(`13px;`)
	}
	sb.WriteString(`
            display: flex;
            justify-content: center;
            align-items: center;
            flex-wrap: wrap;
            gap: 12px;
            text-align: center;
            margin-top: 4px;
        }
        .contact-info span {
            display: inline-flex;
            align-items: center;
        }
        .contact-info svg {
            width: 12px !important;
            height: 12px !important;
            min-width: 12px !important;
            min-height: 12px !important;
            max-width: 12px !important;
            max-height: 12px !important;
            margin-right: 4px;
            vertical-align: -1px;
            fill: #000;
            flex-shrink: 0;
            display: inline-block;
        }

        /* Section Headings */
        h2 {
            font-family: "Times New Roman", Times, serif;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`14.5px; margin-top: 8px; margin-bottom: 5px;`)
	} else {
		sb.WriteString(`15.5px; margin-top: 12px; margin-bottom: 6px;`)
	}
	sb.WriteString(`
            text-transform: uppercase;
            border-bottom: 1.5px solid #000;
            border-top: none;
            padding-bottom: 2px;
            font-weight: bold;
        }

        /* General Content Styling */
        p {
            font-family: "Times New Roman", Times, serif;
            margin: 0 0 8px 0;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13px;`)
	} else {
		sb.WriteString(`13.5px;`)
	}
	sb.WriteString(`
            text-align: justify;
        }
        
        .flex-between {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
        }

        /* Work Experience Styling */
        .job-title-container {
            margin-bottom: 1px;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13.5px;`)
	} else {
		sb.WriteString(`14.5px;`)
	}
	sb.WriteString(`
        }
        .job-title {
            font-family: "Times New Roman", Times, serif;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13.5px;`)
	} else {
		sb.WriteString(`14.5px;`)
	}
	sb.WriteString(`
        }
        .job-title strong {
            font-weight: bold;
        }
        .job-date {
            font-family: "Times New Roman", Times, serif;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13px;`)
	} else {
		sb.WriteString(`13.5px;`)
	}
	sb.WriteString(`
        }
        .company-container {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13px; margin-bottom: 3px;`)
	} else {
		sb.WriteString(`13.5px; margin-bottom: 4px;`)
	}
	sb.WriteString(`
        }
        .company-name, .job-location {
            font-style: italic;
        }

        ul {
            font-family: "Times New Roman", Times, serif;
            margin: 0 0 4px 0;
            padding-left: 20px;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13px;`)
	} else {
		sb.WriteString(`13.5px;`)
	}
	sb.WriteString(`
            text-align: justify;
            list-style-type: disc !important;
        }
        li {
            font-family: "Times New Roman", Times, serif;
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`2px; line-height: 1.2;`)
	} else {
		sb.WriteString(`3px; line-height: 1.28;`)
	}
	sb.WriteString(`
            list-style-type: disc !important;
        }
        li strong {
            font-weight: bold;
        }

        .tech-used {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`12.5px; margin-top: 3px; margin-bottom: 6px;`)
	} else {
		sb.WriteString(`13px; margin-top: 3px; margin-bottom: 10px;`)
	}
	sb.WriteString(`
        }
        .tech-used em {
            font-style: italic;
        }

        /* Education */
        .edu-details {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13px; margin-top: 1px; margin-bottom: 5px;`)
	} else {
		sb.WriteString(`13.5px; margin-top: 1px; margin-bottom: 5px;`)
	}
	sb.WriteString(`
        }

        /* Skills Table */
        .skills-table {
            font-family: "Times New Roman", Times, serif;
            width: 100%;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`13px;`)
	} else {
		sb.WriteString(`13.5px;`)
	}
	sb.WriteString(`
            border-collapse: collapse;
            margin-bottom: 6px;
        }
        .skills-table td {
            vertical-align: top;
            padding: 2.5px 0;
        }
        .skills-table td:first-child {
            font-weight: bold;
            width: 26%;
            padding-right: 8px;
        }
        @media print {
            body { padding: 0; margin: 0; max-width: 100%; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
            .job-title-container, .company-container, section { page-break-inside: avoid; }
        }
    </style>
</head>
<body>

    <header>
        <h1>`)
	sb.WriteString(html.EscapeString(sr.Name))
	sb.WriteString(`</h1>`)
	if sr.Title != "" {
		sb.WriteString(`
        <div class="subtitle"><em>`)
		sb.WriteString(html.EscapeString(sr.Title))
		sb.WriteString(`</em></div>`)
	}
	sb.WriteString(`
        <div class="contact-info">`)
	for _, item := range sr.ContactItems {
		cleanItem := strings.TrimSpace(strings.ReplaceAll(item, "|", ""))
		if cleanItem == "" {
			continue
		}
		sb.WriteString(`
            <span>`)
		svgStyle := `width="12" height="12" style="width:12px!important;height:12px!important;min-width:12px!important;min-height:12px!important;max-width:12px!important;max-height:12px!important;vertical-align:-1px;margin-right:4px;fill:#000;display:inline-block;flex-shrink:0;"`
		if strings.Contains(cleanItem, "@") {
			sb.WriteString(fmt.Sprintf(`<svg %s viewBox="0 0 512 512"><path d="M48 64C21.5 64 0 85.5 0 112c0 15.1 7.1 29.3 19.2 38.4L236.8 313.6c11.4 8.5 27 8.5 38.4 0L492.8 150.4c12.1-9.1 19.2-23.3 19.2-38.4c0-26.5-21.5-48-48-48H48zM0 176V384c0 35.3 28.7 64 64 64H448c35.3 0 64-28.7 64-64V176L294.4 339.2c-22.8 17.1-54 17.1-76.8 0L0 176z"/></svg>`, svgStyle))
		} else if regexp.MustCompile(`[\+\d\(\)\-]{7,}`).MatchString(cleanItem) {
			sb.WriteString(fmt.Sprintf(`<svg %s viewBox="0 0 512 512"><path d="M164.9 24.6c-7.7-18.6-28-28.5-47.4-23.2l-88 24C12.1 30.2 0 46 0 64C0 311.4 200.6 512 448 512c18 0 33.8-12.1 38.6-29.5l24-88c5.3-19.4-4.6-39.7-23.2-47.4l-96-40c-16.3-6.8-35.2-2.1-46.3 11.6L304.7 368C234.3 334.7 177.3 277.7 144 207.3L193.3 167c13.7-11.2 18.4-30 11.6-46.3l-40-96z"/></svg>`, svgStyle))
		} else {
			sb.WriteString(fmt.Sprintf(`<svg %s viewBox="0 0 384 512"><path d="M215.7 499.2C267 435 384 279.4 384 192C384 86 298 0 192 0S0 86 0 192c0 87.4 117 243 168.3 307.2c12.3 15.3 35.1 15.3 47.4 0zM192 128a64 64 0 1 1 0 128 64 64 0 1 1 0-128z"/></svg>`, svgStyle))
		}
		sb.WriteString(html.EscapeString(cleanItem))
		sb.WriteString(`</span>`)
	}
	sb.WriteString(`
        </div>
    </header>`)

	if sr.Summary != "" {
		sb.WriteString(`

    <section>
        <h2>PROFESSIONAL SUMMARY</h2>
        <p>`)
		sb.WriteString(renderFormattedTextGo(sr.Summary))
		sb.WriteString(`</p>
    </section>`)
	}

	if len(sr.WorkExperience) > 0 {
		sb.WriteString(`

    <section>
        <h2>WORK EXPERIENCES</h2>`)
		for _, job := range sr.WorkExperience {
			sb.WriteString(`
        <div class="job-title-container flex-between">
            <div class="job-title">`)
			sb.WriteString(formatJobTitleLineGo(job.Title))
			sb.WriteString(`</div>
            <div class="job-date">`)
			sb.WriteString(html.EscapeString(job.Date))
			sb.WriteString(`</div>
        </div>`)
			if job.Company != "" || job.Location != "" {
				sb.WriteString(`
        <div class="company-container flex-between">
            <div class="company-name"><em>`)
				sb.WriteString(renderFormattedTextGo(job.Company))
				sb.WriteString(`</em></div>
            <div class="job-location"><em>`)
				sb.WriteString(renderFormattedTextGo(job.Location))
				sb.WriteString(`</em></div>
        </div>`)
			}
			if len(job.Bullets) > 0 {
				sb.WriteString(`
        <ul>`)
				for _, b := range job.Bullets {
					sb.WriteString(`
            <li>`)
					sb.WriteString(formatBulletActionVerbGo(b))
					sb.WriteString(`</li>`)
				}
				sb.WriteString(`
        </ul>`)
			}
			if job.TechStack != "" {
				sb.WriteString(`
        <div class="tech-used"><em>Technologies / Skills Used : `)
				sb.WriteString(renderFormattedTextGo(job.TechStack))
				sb.WriteString(`</em></div>`)
			}
		}
		sb.WriteString(`
    </section>`)
	}

	if len(sr.Education) > 0 {
		fontSize := "14.5px"
		if isSinglePage {
			fontSize = "13.5px"
		}
		sb.WriteString(`

    <section>
        <h2>EDUCATIONS</h2>`)
		for _, edu := range sr.Education {
			sb.WriteString(fmt.Sprintf(`
        <div class="flex-between" style="font-size: %s; font-family: 'Times New Roman', Times, serif;">
            <div><strong>%s</strong></div>
            <div>%s</div>
        </div>
        <div class="edu-details"><em>%s</em></div>`,
				fontSize,
				renderFormattedTextGo(edu.Institution),
				html.EscapeString(edu.Date),
				renderFormattedTextGo(edu.Degree),
			))
		}
		sb.WriteString(`
    </section>`)
	}

	if len(sr.Skills) > 0 {
		sb.WriteString(`

    <section>
        <h2>SKILLS</h2>
        <table class="skills-table">`)
		for _, skill := range sr.Skills {
			cat := strings.TrimSpace(strings.TrimSuffix(skill.Category, ":"))
			sb.WriteString(fmt.Sprintf(`
            <tr>
                <td><strong>%s :</strong></td>
                <td>%s</td>
            </tr>`,
				renderFormattedTextGo(cat),
				renderFormattedTextGo(skill.Items),
			))
		}
		sb.WriteString(`
        </table>
    </section>`)
	}

	sb.WriteString(`

</body>
</html>`)

	return sb.String()
}

// ValidateResumeWithRecruiter acts as an independent Senior Technical Recruiter auditing the generated resume against original source resume text
func (c *Client) ValidateResumeWithRecruiter(originalText string, generated *models.StructuredResume, modelOverride string) (*models.RecruiterValidationResult, error) {
	if generated == nil {
		return nil, fmt.Errorf("generated resume is nil")
	}

	genJSON, err := json.MarshalIndent(generated, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal generated resume: %w", err)
	}

	prompt := fmt.Sprintf(`You are a Senior Executive Recruiter and Resume Verifier.
Your task is to stringently audit an AI-generated/tailored resume against the candidate's ORIGINAL resume source text.

Verify with 100%% precision:
1. HALLUCINATIONS: Check if the generated resume contains any fabricated companies, fictitious job titles, degrees, or certifications that DO NOT exist in the original source resume text.
2. OMISSIONS: Check if any real companies, employment history, or education entries from the original source resume were wrongfully deleted or omitted.
3. DUMMY DATA: Check if there is any placeholder or dummy text (e.g. "Lorem Ipsum", "N/A", "Jane Doe", "[Company Name]", "TBD", "Filler", etc.).
4. QUALITY ASSESSMENT: Rate the tailored resume from a recruiter's perspective (0-100 score).

Respond ONLY with a valid JSON object matching this exact schema:
{
  "is_valid": true,
  "hallucinations": [],
  "omissions": [],
  "dummy_data": [],
  "quality_feedback": "<Detailed recruiter audit notes>",
  "recruiter_score": 95
}

CRITICAL INSTRUCTIONS:
- If NO hallucinations, omissions, or dummy data exist, return empty arrays [] for those fields and set "is_valid": true.
- If any real company from the original resume is missing, list it in "omissions" and set "is_valid": false.
- If any fake company/title/degree is added, list it in "hallucinations" and set "is_valid": false.

ORIGINAL SOURCE RESUME TEXT:
%s

GENERATED STRUCTURED RESUME JSON:
%s

OUTPUT STRICTLY VALID JSON:`, truncate(originalText, 500000), string(genJSON))

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
