package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"remotehunter/internal/models"
	"strings"
	"testing"
)

const brightVisionJobDescription = `
Bright Vision Technologies is a forward-thinking software development company dedicated to building innovative solutions that help businesses automate and optimize their operations.
Golang Developer
Job Title: Golang Developer
Location: 100% Remote (Continental United States)
Salary: $100K - $150K / Annum
Experience: 5+ years
Employment Type: Full-time, direct W2 with Bright Vision Technologies

Job Summary:
We are seeking an experienced Golang Developer to design and build high-performance backend services, infrastructure tooling, and cloud-native applications using Go. In this role you will work on systems where latency, concurrency, and operational efficiency are first-class concerns, and you will contribute to a codebase shared by engineers across multiple teams. The ideal candidate will combine strong Go expertise with broad systems knowledge, including network programming, container ecosystems, and distributed system design.

Key Responsibilities:
- Design and implement performant backend services and APIs in Go, with strong attention to concurrency, error handling, and resource management
- Build cloud-native applications using Go and Kubernetes-native libraries, applying idiomatic Go patterns, well-defined module boundaries, and operational hooks
- Develop CLI tools, Kubernetes controllers, and custom operators for internal platforms
- Implement gRPC and REST APIs with appropriate observability and security
- Profile and optimize Go applications for memory, GC, and goroutine behavior
- Integrate with messaging systems (Kafka, NATS) and data stores (PostgreSQL, Redis, etcd)
- Build comprehensive automated tests, including unit, integration, and benchmark tests
- Implement structured logging, metrics emission, and distributed tracing throughout services

Required Qualifications:
- Bachelor’s degree in Computer Science, Engineering, or a related technical discipline
- 5 or more years of professional software engineering experience, with significant time in Go
- Strong understanding of Go concurrency patterns (goroutines, channels, contexts)
- Hands-on experience building production gRPC and/or REST APIs in Go
- Experience with Kubernetes-native development (client-go, controller-runtime)
- Solid experience with relational and key-value data stores
- Strong understanding of distributed systems and networking fundamentals
- Experience with CI/CD pipelines and container-based deployment
- Excellent debugging and performance-engineering skills

Preferred Qualifications:
- Open-source contributions to Go-based projects
- Experience writing Kubernetes operators or controllers
- Familiarity with eBPF, service mesh, or low-level systems programming
- Exposure to security-sensitive Go projects (cryptography, auth)
- Experience with WASM, Tinygo, or embedded systems
`

const sourceResumeText = `Venugopal Hegde
Senior Software Development Engineer
Hosagadde, Sirsi, Karnataka 581318 | +91 9632968298 | venuhegde6@gmail.com

CAREER SUMMARY
Senior Software Development Engineer with 7+ years of experience in architecting scalable backend systems and cloud-native platforms. Adept at working independently and collaboratively in Agile environments, I leverage my expertise in Go and Kubernetes, alongside advanced AI tools, to deliver innovative, efficient solutions.

TECHNICAL SKILLS
Languages: Go (Golang), Python, TypeScript, SQL, Shell Scripting
Cloud, Automation & Infrastructure: Kubernetes (GKE), Docker, Helm, AWS (Lambda, API Gateway, ECS), CI/CD Automation
Data & Messaging: PostgreSQL, DynamoDB, Redis, MongoDB, NATS, Google Pub/Sub, gRPC, WebSockets
Practices: Platform Engineering, Test-Driven Development (TDD), System Design, Agile/Scrum, Remote Collaboration

PROFESSIONAL EXPERIENCE
EPAM Systems | Bangalore, India
Backend Developer (FDPlan - Schlumberger) Jun 2023 - Present
• Architected the decoupling of a heavily loaded backend system into two specialized microservices (dedicated REST API and WebSocket/Pub/Sub handlers), eliminating intense CPU spikes and optimizing resource allocation.
• Autonomously resolved critical, intermittent production bugs related to Google Pub/Sub channel rate limits by discovering and optimizing topic creation, significantly reducing active streams.
• Executed this complex architectural migration with zero downtime, ensuring uninterrupted service.
• Optimized middleware resource services, implementing Redis caching to reduce datastore load and improve API response times by 30%.
• Leveraged AI tools to automate routine coding tasks and generate test cases.
• Integrated the SLB OSDU Data Platform for enterprise modules, collaborating with cross-functional remote teams.
Technologies / Skills Used : Go, Kubernetes, Google Pub/Sub, Redis

EPAM Systems | Bangalore, India
Portability Engineer (Client: Schlumberger) Jun 2021 - May 2023
• Developed automation tooling in Go to accelerate Docker-based infrastructure provisioning and migration.
• Built and deployed multi-cloud storage libraries (AWS, Azure, GCP), defining production-ready portability patterns.
• Championed Test-Driven Development (TDD) and automated testing strategies, increasing code coverage to 95%.
• Collaborated with remote, cross-functional teams to define observability standards and enforce system resilience using Kubernetes best practices.
Technologies / Skills Used : Go, Docker, AWS, Azure, GCP

InTimeTec | Bangalore, India
GoLang Developer (Client: KOUNT) Jun 2019 - May 2021
• Built robust microservices for advanced fraud protection, integrating third-party APIs for automated dispute detection.
• Automated internal communications and deployments using gRPC and Kubernetes, ensuring seamless scaling for high-volume traffic events.
• Owned and improved CI/CD workflows, enabling weekly releases and reducing manual deployment time by 75%.
• Led API automation initiatives using Mocha, significantly improving test coverage and reducing rollout bugs.
Technologies / Skills Used : Go, gRPC, Kubernetes

EDUCATION
Impact College of Engineering & Applied Sciences | Bangalore, India
Bachelor of Engineering in Electronics & Communications 2015 - 2019
`

func TestBrightVisionZeroExperienceLoss(t *testing.T) {
	// Test mock AI server returning structured resume
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{
							"name": "Venugopal Hegde",
							"title": "Senior Software Development Engineer",
							"contact_items": ["+91 9632968298", "venuhegde6@gmail.com", "Hosagadde, India"],
							"summary": "Senior Software Development Engineer with 7+ years of experience in Go and Kubernetes.",
							"work_experience": [
								{
									"title": "Backend Developer (FDPlan - Schlumberger)",
									"company": "EPAM Systems",
									"date": "Jun 2023 - Present",
									"location": "Bangalore, India",
									"bullets": ["Architected decoupling of backend services in Go."],
									"tech_stack": "Go, Kubernetes, Google Pub/Sub, Redis"
								},
								{
									"title": "Portability Engineer (Client: Schlumberger)",
									"company": "EPAM Systems",
									"date": "Jun 2021 - May 2023",
									"location": "Bangalore, India",
									"bullets": ["Developed Go automation tooling."],
									"tech_stack": "Go, Docker, AWS, Azure, GCP"
								},
								{
									"title": "GoLang Developer (Client: KOUNT)",
									"company": "InTimeTec",
									"date": "Jun 2019 - May 2021",
									"location": "Bangalore, India",
									"bullets": ["Built microservices in Go for fraud protection."],
									"tech_stack": "Go, gRPC, Kubernetes"
								}
							],
							"education": [
								{
									"institution": "Impact College of Engineering & Applied Sciences",
									"date": "2015 - 2019",
									"degree": "Bachelor of Engineering in Electronics & Communications"
								}
							],
							"skills": [
								{"category": "Databases", "items": "PostgreSQL, DynamoDB, Redis, MongoDB"},
								{"category": "Frameworks & Libraries", "items": "Kubernetes, Docker, Helm"},
								{"category": "Programming Languages", "items": "Go, Python, TypeScript, SQL"},
								{"category": "Soft Skills", "items": "Platform Engineering, TDD, System Design"},
								{"category": "Tools & Platforms", "items": "AWS, CI/CD Automation"}
							]
						}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("mock-key", server.URL, "z-ai/glm-5.2")

	structRes, htmlContent, err := client.ConvertResumeToTemplate(sourceResumeText, "", true)
	if err != nil {
		t.Fatalf("ConvertResumeToTemplate failed: %v", err)
	}
	_ = htmlContent

	if len(structRes.WorkExperience) != 3 {
		genBytes, _ := json.MarshalIndent(structRes, "", "  ")
		t.Logf("DEBUG: Raw LLM Output:\n%s", string(genBytes))
		t.Fatalf("Expected exactly 3 work experience entries (ZERO loss), got %d", len(structRes.WorkExperience))
	}

	// Verify all company names exist directly on unmarshaled Go struct
	expectedCompanies := map[string]bool{"EPAM Systems": false, "InTimeTec": false}
	for _, job := range structRes.WorkExperience {
		for comp := range expectedCompanies {
			if strings.Contains(job.Company, comp) {
				expectedCompanies[comp] = true
			}
		}
	}

	for comp, found := range expectedCompanies {
		if !found {
			t.Errorf("Expected structured output to contain company '%s'", comp)
		}
	}

	// Verify all job titles exist
	titles := []string{"Backend Developer", "Portability Engineer", "GoLang Developer"}
	for _, title := range titles {
		found := false
		for _, job := range structRes.WorkExperience {
			if strings.Contains(job.Title, title) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected work experience to contain job title '%s'", title)
		}
	}

	// Verify skill categories exist
	if len(structRes.Skills) < 4 {
		t.Errorf("Expected at least 4 skill categories, got %d", len(structRes.Skills))
	}
}

func TestBrightVisionPromptChainingIntegration(t *testing.T) {
	apiKey := os.Getenv("NVIDIA_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping live NVIDIA integration test: NVIDIA_API_KEY environment variable not set")
		return
	}
	baseURL := os.Getenv("NVIDIA_BASE_URL")
	model := os.Getenv("NVIDIA_MODEL")

	client := NewClient(apiKey, baseURL, model)

	job := &models.Job{
		ID:          "bright-vision-1",
		Title:       "Golang Developer",
		Company:     "Bright Vision Technologies",
		Description: brightVisionJobDescription,
	}

	res := &models.Resume{
		ID:              "resume-venugopal",
		ExtractedSkills: []string{"Go", "Kubernetes", "gRPC", "REST", "Redis", "PostgreSQL", "Kafka", "NATS", "Docker", "TDD", "CI/CD"},
		RawText:         sourceResumeText,
	}

	// Step 1: Initial ATS Match Analysis
	t.Log("Step 1: Running initial ATS analysis against Bright Vision Golang Developer role...")
	initialAnalysis, err := client.AnalyzeATSMatch(job, res, "")
	if err != nil {
		t.Fatalf("Initial ATS analysis failed: %v", err)
	}
	t.Logf("Initial ATS Score: %d / 100", initialAnalysis.ATSScore)

	// Step 2: Convert resume to single-page structured format with ZERO work experience loss
	t.Log("Step 2: Converting resume to single page template via AI...")
	structRes, htmlContent, err := client.ConvertResumeToTemplate(sourceResumeText, "", true)
	if err != nil {
		t.Fatalf("ConvertResumeToTemplate failed: %v", err)
	}

	if len(structRes.WorkExperience) != 3 {
		genBytes, _ := json.MarshalIndent(structRes, "", "  ")
		t.Logf("DEBUG: Raw LLM Output:\n%s\nHTML:\n%s", string(genBytes), htmlContent)
		t.Fatalf("CRITICAL ASSERTION FAILED: Single page fit stripped job entries! Expected 3 experiences, got %d", len(structRes.WorkExperience))
	} else {
		t.Logf("PASSED: All %d work experience entries preserved cleanly.", len(structRes.WorkExperience))
	}

	// Step 3: Run ChatWithResume prompt chaining to tailor bullets for Bright Vision role
	t.Log("Step 3: Running ChatWithResume prompt chaining to tailor resume for Bright Vision...")
	chatReq := &models.ChatRequest{
		ResumeText: sourceResumeText,
		Message:    "Tailor my resume for the Bright Vision Golang Developer role. Focus on Go concurrency, gRPC, REST, Kubernetes client-go, PostgreSQL, Redis, Kafka, NATS, and TDD while preserving all 3 of my work experiences.",
	}
	chatResp, err := client.ChatWithResume(chatReq, brightVisionJobDescription, "")
	if err != nil {
		t.Fatalf("ChatWithResume prompt chaining failed: %v", err)
	}

	if chatResp.StructuredResume == nil && len(chatResp.GapPrompts) > 0 {
		t.Logf("Phase 1 gap prompts received (%d questions). Sending candidate responses to trigger Phase 2 tailored resume...", len(chatResp.GapPrompts))
		phase2Req := &models.ChatRequest{
			ResumeText: sourceResumeText,
			Message:    "I have hands-on experience with all these gaps: I have built production Go concurrency systems with goroutines and channels, used Kubernetes client-go in EPAM, built high-performance gRPC and REST APIs in Go, and managed distributed systems with Docker and CI/CD. Please generate the full tailored structured_resume now with 90%+ ATS match.",
		}
		chatResp, err = client.ChatWithResume(phase2Req, brightVisionJobDescription, "")
		if err != nil {
			t.Fatalf("Phase 2 ChatWithResume failed: %v", err)
		}
	}

	if chatResp.StructuredResume != nil {
		t.Logf("Prompt Chaining Output: Generated structured resume with %d jobs", len(chatResp.StructuredResume.WorkExperience))
		if len(chatResp.StructuredResume.WorkExperience) < 3 {
			genBytes, _ := json.MarshalIndent(chatResp.StructuredResume, "", "  ")
			t.Logf("DEBUG: Raw LLM Output:\n%s", string(genBytes))
			t.Errorf("CRITICAL ASSERTION FAILED: Chat prompt chaining stripped work experiences! Expected 3, got %d", len(chatResp.StructuredResume.WorkExperience))
		}
	}

	// Step 4: Re-evaluate tailored resume ATS Match Score
	tailoredText := sourceResumeText
	if chatResp.FullResumeReplacement != "" {
		tailoredText = chatResp.FullResumeReplacement
	} else if chatResp.StructuredResume != nil {
		tailoredText = BuildATSTemplateHTML(chatResp.StructuredResume, true)
	} else if htmlContent != "" {
		tailoredText = htmlContent
	}

	tailoredResume := &models.Resume{
		ID:              "tailored-resume",
		ExtractedSkills: res.ExtractedSkills,
		RawText:         tailoredText,
	}

	t.Log("Step 4: Re-evaluating tailored resume ATS score...")
	finalAnalysis, err := client.AnalyzeATSMatch(job, tailoredResume, "")
	if err != nil {
		t.Fatalf("Final ATS analysis failed: %v", err)
	}

	t.Logf("Final ATS Score after Prompt Chaining: %d / 100", finalAnalysis.ATSScore)
	if finalAnalysis.ATSScore < initialAnalysis.ATSScore {
		t.Errorf("PROMPT CHAINING FAILED: Tailored ATS Score (%d) is lower than Initial ATS Score (%d)", 
			finalAnalysis.ATSScore, initialAnalysis.ATSScore)
	}

	// Step 5: Validate Generated Resume with Independent Senior Recruiter AI Audit
	t.Log("Step 5: Passing generated content to Senior Recruiter AI role for audit & validation...")
	genToAudit := structRes
	if chatResp.StructuredResume != nil {
		genToAudit = chatResp.StructuredResume
	}

	recruiterAudit, err := client.ValidateResumeWithRecruiter(sourceResumeText, genToAudit, "")
	if err != nil {
		t.Fatalf("ValidateResumeWithRecruiter failed: %v", err)
	}

	t.Logf("Recruiter AI Audit Score: %d / 100", recruiterAudit.RecruiterScore)
	t.Logf("Recruiter Feedback: %s", recruiterAudit.QualityFeedback)

	const minimumAcceptableScore = 80
	if recruiterAudit.RecruiterScore < minimumAcceptableScore {
		t.Errorf("RECRUITER AUDIT FAILED: Score %d is below the minimum threshold of %d. Feedback: %s", 
			recruiterAudit.RecruiterScore, minimumAcceptableScore, recruiterAudit.QualityFeedback)
	}

	if len(recruiterAudit.Hallucinations) > 0 {
		t.Errorf("RECRUITER AUDIT FAILED: Detected hallucinated data: %v", recruiterAudit.Hallucinations)
	}
	if len(recruiterAudit.Omissions) > 0 {
		t.Errorf("RECRUITER AUDIT FAILED: Detected omitted work experiences: %v", recruiterAudit.Omissions)
	}
	if len(recruiterAudit.DummyData) > 0 {
		t.Errorf("RECRUITER AUDIT FAILED: Detected dummy data: %v", recruiterAudit.DummyData)
	}
	if !recruiterAudit.IsValid {
		t.Errorf("RECRUITER AUDIT FAILED: Recruiter marked resume as invalid!")
	} else {
		t.Log("PASSED: Senior Recruiter AI verified ZERO hallucinations, ZERO omissions, and ZERO dummy data!")
	}

	_ = htmlContent
}

func TestRecruiterValidationMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{
							"is_valid": true,
							"hallucinations": [],
							"omissions": [],
							"dummy_data": [],
							"quality_feedback": "The resume accurately reflects candidate Venugopal Hegde's original 3 work experiences with zero dummy text or fake claims.",
							"recruiter_score": 98
						}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("mock-key", server.URL, "z-ai/glm-5.2")

	dummyGen := &models.StructuredResume{
		Name:  "Venugopal Hegde",
		Title: "Senior Software Development Engineer",
		WorkExperience: []models.JobExperience{
			{Title: "Backend Developer", Company: "EPAM Systems"},
			{Title: "Portability Engineer", Company: "EPAM Systems"},
			{Title: "GoLang Developer", Company: "InTimeTec"},
		},
	}

	audit, err := client.ValidateResumeWithRecruiter(sourceResumeText, dummyGen, "")
	if err != nil {
		t.Fatalf("ValidateResumeWithRecruiter failed: %v", err)
	}

	if !audit.IsValid {
		t.Errorf("Expected valid audit, got invalid")
	}
	if audit.RecruiterScore != 98 {
		t.Errorf("Expected recruiter score 98, got %d", audit.RecruiterScore)
	}
	if len(audit.Hallucinations) != 0 || len(audit.Omissions) != 0 || len(audit.DummyData) != 0 {
		t.Errorf("Expected 0 hallucinations/omissions/dummy_data, got %v / %v / %v", audit.Hallucinations, audit.Omissions, audit.DummyData)
	}
}
