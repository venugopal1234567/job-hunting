package ai

import (
	"os"
	"remotehunter/internal/models"
	"testing"
)

func TestAnalyzeATSMatchIntegration(t *testing.T) {
	apiKey := os.Getenv("NVIDIA_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: NVIDIA_API_KEY not set")
		return
	}
	baseURL := os.Getenv("NVIDIA_BASE_URL")
	model := os.Getenv("NVIDIA_MODEL")

	client := NewClient(apiKey, baseURL, model)

	job := &models.Job{
		ID:          "job-integration-test",
		Title:       "Senior Go Backend Engineer",
		Company:     "Acme Corp",
		Description: "We are looking for a Senior Go Developer. Must have strong skills in Go, PostgreSQL, Docker, and Kubernetes.",
	}

	// 1. Initial Resume
	resume1 := &models.Resume{
		ID:              "resume-integration-test-1",
		ExtractedSkills: []string{"Go", "Docker"},
		RawText:         "Venugopal Hegde. Software Engineering Senior Engineer. Skills: Go, Docker, Python.",
	}

	analysis1, err := client.AnalyzeATSMatch(job, resume1, "")
	if err != nil {
		t.Fatalf("AnalyzeATSMatch failed: %v", err)
	}

	t.Logf("[Integration] Run 1 Score: %d", analysis1.ATSScore)

	// 2. Repeat to verify consistency (determinism at temperature 0.0)
	analysis1Repeat, err := client.AnalyzeATSMatch(job, resume1, "")
	if err != nil {
		t.Fatalf("Repeat AnalyzeATSMatch failed: %v", err)
	}

	t.Logf("[Integration] Run 2 (Repeat) Score: %d", analysis1Repeat.ATSScore)
	if analysis1.ATSScore != analysis1Repeat.ATSScore {
		t.Errorf("Inconsistent real AI scores under temperature 0.0: first = %d, repeat = %d", analysis1.ATSScore, analysis1Repeat.ATSScore)
	}

	// 3. Edited Resume (applying suggestions by adding missing skills)
	resume2 := &models.Resume{
		ID:              "resume-integration-test-2",
		ExtractedSkills: []string{"Go", "Docker", "PostgreSQL", "Kubernetes"},
		RawText:         "Venugopal Hegde. Software Engineering Senior Engineer. Skills: Go, Docker, PostgreSQL, Kubernetes, Python.",
	}

	analysis2, err := client.AnalyzeATSMatch(job, resume2, "")
	if err != nil {
		t.Fatalf("AnalyzeATSMatch with edited resume failed: %v", err)
	}

	t.Logf("[Integration] Run 3 (Edited) Score: %d", analysis2.ATSScore)
	if analysis2.ATSScore <= analysis1.ATSScore {
		t.Errorf("Expected real AI score to increase when missing skills are added. Initial: %d, Edited: %d", analysis1.ATSScore, analysis2.ATSScore)
	}
}
