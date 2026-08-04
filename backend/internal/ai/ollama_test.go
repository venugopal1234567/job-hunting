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
	// Create mock HTTP server representing Ollama
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method and path
		if r.Method != http.MethodPost || r.URL.Path != "/api/generate" {
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Decode request body
		var req ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify options (specifically, temperature must be 0.0)
		temp, exists := req.Options["temperature"]
		if !exists {
			t.Errorf("Expected temperature option to be set")
		} else if tempVal, ok := temp.(float64); !ok || tempVal != 0.0 {
			t.Errorf("Expected temperature option to be 0.0, got: %v", temp)
		}

		// Send mock JSON response
		resp := ollamaResponse{
			Response: `{"ats_score": 85, "matched_skills": ["Go", "Docker"], "missing_skills": ["Kubernetes"], "actionable_suggestions": ["Add Kubernetes"], "gap_questions": []}`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "gemma")
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

	t.Run("Ollama Host Unavailable", func(t *testing.T) {
		// Call with an invalid/closed host
		client := NewClient("http://localhost:9999", "gemma")
		_, err := client.AnalyzeATSMatch(job, res, "")
		if err == nil {
			t.Fatal("Expected error when Ollama host is unavailable, got nil")
		}
		if !strings.Contains(err.Error(), "ollama unavailable") {
			t.Errorf("Expected 'ollama unavailable' error, got: %v", err)
		}
	})

	t.Run("Ollama Non-200 Status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL, "gemma")
		_, err := client.AnalyzeATSMatch(job, res, "")
		if err == nil {
			t.Fatal("Expected error when Ollama returns non-200 status, got nil")
		}
		if !strings.Contains(err.Error(), "ollama returned status 500") {
			t.Errorf("Expected status 500 error, got: %v", err)
		}
	})

	t.Run("Ollama Bad JSON Response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ollamaResponse{
				Response: `invalid-json{`,
				Done:     true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(server.URL, "gemma")
		_, err := client.AnalyzeATSMatch(job, res, "")
		if err == nil {
			t.Fatal("Expected error when parsing bad response, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse LLM JSON response") {
			t.Errorf("Expected 'failed to parse' error, got: %v", err)
		}
	})
}
