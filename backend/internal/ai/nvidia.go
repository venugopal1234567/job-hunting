package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
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

// generateCompletion routes requests to OpenAI/NVIDIA API endpoint
func (c *Client) generateCompletion(prompt string, modelOverride string, jsonFormat bool) (string, error) {
	model := c.resolveModel(modelOverride)
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

	maxTokens := 131072

	reqObj := openAIReq{
		Model: model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens:   maxTokens,
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
		return "", fmt.Errorf("nvidia api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[AI Client] NVIDIA API Error Status %d (endpoint %s, model %s): %s", resp.StatusCode, endpoint, model, string(bodyBytes))
		return "", fmt.Errorf("nvidia api returned status %d for model '%s' at endpoint '%s': %s", resp.StatusCode, model, endpoint, string(bodyBytes))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", fmt.Errorf("decode nvidia response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("empty response choices from nvidia api")
	}
	return openAIResp.Choices[0].Message.Content, nil
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

CRITICAL RULES:
- SECTION ORDERING IS MANDATORY: 1. Professional Summary, 2. Skills (placed directly below summary!), 3. Work Experiences, 4. Educations.
- The "highlight_keywords" array MUST contain all high-value keywords, technologies, skills, and metrics that should be highlighted on the resume for maximum ATS/recruiter impact.
- All bullet points in work_experience MUST be strong, concise, and impact-oriented sentences.
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
		modelOverride = "nvidia/nemotron-3.5-lightning-30b-a3b"
	}

	fitInstruction := ""
	if fitSinglePage {
		fitInstruction = "\nCRITICAL SINGLE PAGE FIT INSTRUCTION: Condense all bullet points into punchy, high-impact single-line achievements. Eliminate filler words and redundant phrases so the entire resume easily fits strictly on 1 single letter page."
	}

	prompt := fmt.Sprintf(`You are an expert ATS resume parser and formatter.
Extract all information from the candidate's resume and return it strictly as a JSON object matching this exact schema:

{
  "name": "<Candidate Full Name>",
  "title": "<Candidate Current / Target Job Title>",
  "contact_items": ["<Phone Number>", "<Email Address>", "<Location (City, Country)>"],
  "summary": "<Professional summary paragraph>",
  "work_experience": [
    {
      "title": "<Job Role / Title>",
      "date": "<Start Date - End Date>",
      "company": "<Company Name - Employment Type>",
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

// BuildATSTemplateHTML generates high-fidelity HTML matching the requested template format and styling
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
		sb.WriteString(`0.25in 0.35in;`)
	} else {
		sb.WriteString(`0.5in 0.5in;`)
	}
	sb.WriteString(`
        }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: `)
	if isSinglePage {
		sb.WriteString(`1.28; font-size: 8.8pt; padding: 10px 15px;`)
	} else {
		sb.WriteString(`1.6; font-size: 10.5pt; padding: 40px 20px;`)
	}
	sb.WriteString(`
            color: #333;
            max-width: 850px;
            margin: 0 auto;
            background: #fff;
        }
        header {
            text-align: left;
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`10px; padding-bottom: 6px;`)
	} else {
		sb.WriteString(`18px; padding-bottom: 12px;`)
	}
	sb.WriteString(`
            border-bottom: 2px solid #2c3e50;
        }
        h1 {
            font-size: `)
	if isSinglePage {
		sb.WriteString(`1.8em; margin: 0 0 2px 0;`)
	} else {
		sb.WriteString(`2.2em; margin: 0 0 4px 0;`)
	}
	sb.WriteString(`
            color: #2c3e50;
            font-weight: 800;
            letter-spacing: 0.5px;
            text-align: left;
        }
        .contact-info {
            font-size: `)
	if isSinglePage {
		sb.WriteString(`0.88em;`)
	} else {
		sb.WriteString(`0.95em;`)
	}
	sb.WriteString(`
            color: #555;
            text-align: left;
        }
        .contact-info span {
            margin: 0 `)
	if isSinglePage {
		sb.WriteString(`4px;`)
	} else {
		sb.WriteString(`10px;`)
	}
	sb.WriteString(`
        }
        h2 {
            color: #2c3e50;
            border-bottom: 1px solid #ccc;
            padding-bottom: `)
	if isSinglePage {
		sb.WriteString(`2px; margin-top: 8px; margin-bottom: 4px; font-size: 1.05em;`)
	} else {
		sb.WriteString(`5px; margin-top: 30px; font-size: 1.25em;`)
	}
	sb.WriteString(`
            text-transform: uppercase;
        }
        .job {
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`8px;`)
	} else {
		sb.WriteString(`25px;`)
	}
	sb.WriteString(`
        }
        .job-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`2px;`)
	} else {
		sb.WriteString(`5px;`)
	}
	sb.WriteString(`
        }
        .job-title {
            font-size: `)
	if isSinglePage {
		sb.WriteString(`1.02em;`)
	} else {
		sb.WriteString(`1.2em;`)
	}
	sb.WriteString(`
            font-weight: bold;
            color: #34495e;
        }
        .job-date {
            font-weight: bold;
            color: #7f8c8d;
        }
        .company-info {
            display: flex;
            justify-content: space-between;
            font-style: italic;
            color: #555;
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`4px;`)
	} else {
		sb.WriteString(`10px;`)
	}
	sb.WriteString(`
        }
        ul {
            margin-top: 0;
            padding-left: `)
	if isSinglePage {
		sb.WriteString(`16px; margin-bottom: 2px;`)
	} else {
		sb.WriteString(`20px;`)
	}
	sb.WriteString(`
        }
        li {
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`2px;`)
	} else {
		sb.WriteString(`8px;`)
	}
	sb.WriteString(`
            text-align: justify;
        }
        .tech-stack {
            font-weight: bold;
            font-size: `)
	if isSinglePage {
		sb.WriteString(`0.85em; margin-top: 2px;`)
	} else {
		sb.WriteString(`0.9em; margin-top: 10px;`)
	}
	sb.WriteString(`
            color: #2c3e50;
        }
        .skills-list {
            list-style-type: none;
            padding: 0;
        }
        .skills-list li {
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`3px;`)
	} else {
		sb.WriteString(`10px;`)
	}
	sb.WriteString(`
        }
        .skill-category {
            font-weight: bold;
            color: #2c3e50;
            width: 220px;
            display: inline-block;
        }
        .education-block {
            margin-bottom: `)
	if isSinglePage {
		sb.WriteString(`6px;`)
	} else {
		sb.WriteString(`15px;`)
	}
	sb.WriteString(`
        }
        @media print {
            body { padding: 0; margin: 0; max-width: 100%; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
            .job, .education-block, section { page-break-inside: avoid; }
        }
    </style>
</head>
<body>

    <header>
        <h1>`)
	sb.WriteString(html.EscapeString(sr.Name))
	sb.WriteString(`</h1>
        <div class="contact-info">`)
	if sr.Title != "" {
		sb.WriteString(`
            <strong>`)
		sb.WriteString(html.EscapeString(sr.Title))
		sb.WriteString(`</strong><br>`)
	}
	for i, item := range sr.ContactItems {
		if i > 0 {
			sb.WriteString(` | `)
		}
		sb.WriteString(`
            <span>`)
		sb.WriteString(html.EscapeString(item))
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
		sb.WriteString(html.EscapeString(sr.Summary))
		sb.WriteString(`</p>
    </section>`)
	}

	if len(sr.WorkExperience) > 0 {
		sb.WriteString(`

    <section>
        <h2>WORK EXPERIENCES</h2>`)
		for _, job := range sr.WorkExperience {
			sb.WriteString(`

        <div class="job">
            <div class="job-header">
                <span class="job-title">`)
			sb.WriteString(html.EscapeString(job.Title))
			sb.WriteString(`</span>
                <span class="job-date">`)
			sb.WriteString(html.EscapeString(job.Date))
			sb.WriteString(`</span>
            </div>`)
			if job.Company != "" || job.Location != "" {
				sb.WriteString(`
            <div class="company-info">
                <span>`)
				sb.WriteString(html.EscapeString(job.Company))
				sb.WriteString(`</span>
                <span>`)
				sb.WriteString(html.EscapeString(job.Location))
				sb.WriteString(`</span>
            </div>`)
			}
			if len(job.Bullets) > 0 {
				sb.WriteString(`
            <ul>`)
				for _, b := range job.Bullets {
					sb.WriteString(`
                <li>`)
					sb.WriteString(html.EscapeString(b))
					sb.WriteString(`</li>`)
				}
				sb.WriteString(`
            </ul>`)
			}
			if job.TechStack != "" {
				sb.WriteString(`
            <div class="tech-stack">Technologies / Skills Used : `)
				sb.WriteString(html.EscapeString(job.TechStack))
				sb.WriteString(`</div>`)
			}
			sb.WriteString(`
        </div>`)
		}
		sb.WriteString(`
    </section>`)
	}

	if len(sr.Education) > 0 {
		sb.WriteString(`

    <section>
        <h2>EDUCATIONS</h2>`)
		for _, edu := range sr.Education {
			sb.WriteString(`
        <div class="education-block">
            <div class="job-header">
                <span class="job-title">`)
			sb.WriteString(html.EscapeString(edu.Institution))
			sb.WriteString(`</span>
                <span class="job-date">`)
			sb.WriteString(html.EscapeString(edu.Date))
			sb.WriteString(`</span>
            </div>
            <div>`)
			sb.WriteString(html.EscapeString(edu.Degree))
			sb.WriteString(`</div>
        </div>`)
		}
		sb.WriteString(`
    </section>`)
	}

	if len(sr.Skills) > 0 {
		sb.WriteString(`

    <section>
        <h2>SKILLS</h2>
        <ul class="skills-list">`)
		for _, skill := range sr.Skills {
			sb.WriteString(`
            <li><span class="skill-category">`)
			sb.WriteString(html.EscapeString(skill.Category))
			sb.WriteString(` :</span> `)
			sb.WriteString(html.EscapeString(skill.Items))
			sb.WriteString(`</li>`)
		}
		sb.WriteString(`
        </ul>
    </section>`)
	}

	sb.WriteString(`

</body>
</html>`)

	return sb.String()
}

