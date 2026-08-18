package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"remotehunter/internal/ai/prompts"
	"remotehunter/internal/models"
	"strings"
)

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
		fitInstruction = prompts.SinglePageFitConstraint
	}

	cleanText := stripHTMLForPrompt(rawText)
	prompt := fmt.Sprintf(prompts.ConvertResumePromptTemplate, fitInstruction, truncate(cleanText, 500000))

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

// structuredToTextGo converts a StructuredResume back into clean plain text for LLM prompts
func structuredToTextGo(sr *models.StructuredResume) string {
	if sr == nil {
		return ""
	}
	var sb strings.Builder

	if sr.Name != "" {
		sb.WriteString(strings.TrimSpace(sr.Name) + "\n")
	}
	if sr.Title != "" {
		sb.WriteString(strings.TrimSpace(sr.Title) + "\n")
	}
	if len(sr.ContactItems) > 0 {
		sb.WriteString(strings.Join(sr.ContactItems, " | ") + "\n")
	}
	sb.WriteString("\n")

	if sr.Summary != "" {
		sb.WriteString("PROFESSIONAL SUMMARY\n")
		sb.WriteString(strings.TrimSpace(sr.Summary) + "\n\n")
	}

	if len(sr.WorkExperience) > 0 {
		sb.WriteString("WORK EXPERIENCE\n")
		for _, job := range sr.WorkExperience {
			title := strings.TrimSpace(job.Title)
			date := strings.TrimSpace(job.Date)
			if title != "" && date != "" {
				sb.WriteString(fmt.Sprintf("%s | %s\n", title, date))
			} else if title != "" {
				sb.WriteString(title + "\n")
			}

			if job.Company != "" || job.Location != "" {
				compLine := []string{}
				if job.Company != "" {
					compLine = append(compLine, job.Company)
				}
				if job.Location != "" {
					compLine = append(compLine, job.Location)
				}
				sb.WriteString(strings.Join(compLine, " | ") + "\n")
			}

			for _, b := range job.Bullets {
				trimB := strings.TrimSpace(b)
				if trimB != "" {
					sb.WriteString(fmt.Sprintf("• %s\n", trimB))
				}
			}

			if job.TechStack != "" {
				sb.WriteString(fmt.Sprintf("Technologies / Skills Used : %s\n", job.TechStack))
			}
			sb.WriteString("\n")
		}
	}

	if len(sr.Education) > 0 {
		sb.WriteString("EDUCATION\n")
		for _, edu := range sr.Education {
			inst := strings.TrimSpace(edu.Institution)
			date := strings.TrimSpace(edu.Date)
			if inst != "" && date != "" {
				sb.WriteString(fmt.Sprintf("%s | %s\n", inst, date))
			} else if inst != "" {
				sb.WriteString(inst + "\n")
			}
			if edu.Degree != "" {
				sb.WriteString(strings.TrimSpace(edu.Degree) + "\n")
			}
			sb.WriteString("\n")
		}
	}

	if len(sr.Skills) > 0 {
		sb.WriteString("SKILLS\n")
		for _, skill := range sr.Skills {
			cat := strings.TrimSpace(skill.Category)
			items := strings.TrimSpace(skill.Items)
			if cat != "" && items != "" {
				sb.WriteString(fmt.Sprintf("%s : %s\n", cat, items))
			}
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}
