package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestExportResumePDFValidation covers the guards on POST /resume/export-pdf
// that run before any Chromium work: a malformed body or a blank html field
// must be rejected up front rather than handed to the PDF service.
func TestExportResumePDFValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"malformed json rejected", `{"html":`, http.StatusBadRequest},
		{"empty html rejected", `{"html":""}`, http.StatusBadRequest},
		{"whitespace-only html rejected", `{"html":"   \n\t "}`, http.StatusBadRequest},
		{"missing html field rejected", `{}`, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			// A zero-value Handler is enough: none of the rejected cases reach
			// a repository, so the validation guard is all under test here.
			r.POST("/resume/export-pdf", (&Handler{}).ExportResumePDF)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/resume/export-pdf",
				strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(`"error"`)) {
				t.Errorf("rejection body should explain itself, got: %s", w.Body.String())
			}
		})
	}
}