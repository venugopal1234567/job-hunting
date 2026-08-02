package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"remotehunter/internal/config"
	"remotehunter/internal/db"
	"remotehunter/internal/resume"
)

func main() {
	cfg := config.Load()
	database, err := db.Connect(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer database.Close()

	// 1. Fetch raw PDF data from the first resume row where it was stored in raw_text
	var rawPDFText string
	err = database.QueryRow("SELECT raw_text FROM resumes WHERE id = '2e9a56c7-fc65-4a7e-8f20-afaf86be177c'").Scan(&rawPDFText)
	if err != nil {
		log.Fatalf("Failed to fetch first resume: %v", err)
	}

	rawPDFBytes := []byte(rawPDFText)
	fmt.Printf("Fetched raw PDF, size: %d bytes\n", len(rawPDFBytes))

	// 2. Extract clean text using pdftotext CLI
	cmd := exec.Command("pdftotext", "-", "-")
	cmd.Stdin = bytes.NewReader(rawPDFBytes)
	var out bytes.Buffer
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("pdftotext failed: %v (stderr: %s)", err, stderr.String())
	}

	cleanText := out.String()
	fmt.Printf("Extracted text successfully. Length: %d\n", len(cleanText))

	// 3. Extract skills using the internal package
	skills := resume.ExtractSkills(cleanText)
	skillsJSON, _ := json.Marshal(skills)

	// 4. Update the active resume row (6ddaf698-c61a-43cb-b920-b53b8991e7f7)
	res, err := database.Exec(
		"UPDATE resumes SET raw_text = $1, extracted_skills = $2 WHERE id = '6ddaf698-c61a-43cb-b920-b53b8991e7f7'",
		cleanText, string(skillsJSON),
	)
	if err != nil {
		log.Fatalf("Failed to update database: %v", err)
	}
	rowsAffected, _ := res.RowsAffected()
	fmt.Printf("Updated active resume, rows affected: %d\n", rowsAffected)
}
