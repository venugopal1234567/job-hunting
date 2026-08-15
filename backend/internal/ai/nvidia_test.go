package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"remotehunter/internal/models"
	"strings"
	"testing"
)

func TestBuildATSPrompt(t *testing.T) {
	job := &models.Job{
		ID:          "job-1",
		Title:       "Tester",
		Company:     "Nokia",
		Description: "Test Automation",
	}

	resume := &models.Resume{
		ID:              "resume-1",
		ExtractedSkills: []string{"Go"},
		RawText:         "Resume text here",
	}

	prompt := buildATSPrompt(job, resume)

	if !strings.Contains(prompt, "Tester") {
		t.Errorf("Expected prompt to contain job title 'Tester'")
	}
	if !strings.Contains(prompt, "Nokia") {
		t.Errorf("Expected prompt to contain company name 'Nokia'")
	}
	if !strings.Contains(prompt, "Go") {
		t.Errorf("Expected prompt to contain candidate skills 'Go'")
	}
}

func TestAnalyzeATSMatchMock(t *testing.T) {
	// Create mock HTTP server representing OpenAI/NVIDIA API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{"ats_score": 85, "matched_skills": ["Go", "Docker"], "missing_skills": ["Kubernetes"], "actionable_suggestions": ["Add Kubernetes"], "gap_questions": []}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("mock-api-key", server.URL, "z-ai/glm-5.2")
	job := &models.Job{ID: "job-1", Title: "Go Dev", Description: "Go, Kubernetes"}
	res := &models.Resume{ID: "resume-1", ExtractedSkills: []string{"Go", "Docker"}}

	analysis, err := client.AnalyzeATSMatch(job, res, "")
	if err != nil {
		t.Fatalf("AnalyzeATSMatch failed: %v", err)
	}

	if analysis.ATSScore != 85 {
		t.Errorf("Expected ATS score 85, got: %d", analysis.ATSScore)
	}
}

func TestAnalyzeATSMatchErrors(t *testing.T) {
	job := &models.Job{ID: "job-1", Title: "Go Dev", Description: "Go, Kubernetes"}
	res := &models.Resume{ID: "resume-1", ExtractedSkills: []string{"Go", "Docker"}}

	t.Run("NVIDIA Host Unavailable", func(t *testing.T) {
		client := NewClient("mock-key", "http://localhost:9999/v1", "z-ai/glm-5.2")
		_, err := client.AnalyzeATSMatch(job, res, "")
		if err == nil {
			t.Fatal("Expected error when host is unavailable, got nil")
		}
		if !strings.Contains(err.Error(), "nvidia api request failed") {
			t.Errorf("Expected 'nvidia api request failed' error, got: %v", err)
		}
	})

	t.Run("NVIDIA Non-200 Status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient("mock-key", server.URL, "z-ai/glm-5.2")
		_, err := client.AnalyzeATSMatch(job, res, "")
		if err == nil {
			t.Fatal("Expected error when server returns 500 status, got nil")
		}
		if !strings.Contains(err.Error(), "nvidia api returned status 500") {
			t.Errorf("Expected status 500 error, got: %v", err)
		}
	})

	t.Run("NVIDIA Bad JSON Response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"content": `invalid-json{`,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient("mock-key", server.URL, "z-ai/glm-5.2")
		_, err := client.AnalyzeATSMatch(job, res, "")
		if err == nil {
			t.Fatal("Expected error when parsing bad response, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse LLM JSON response") {
			t.Errorf("Expected 'failed to parse' error, got: %v", err)
		}
	})
}
