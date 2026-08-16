package ai

import (
	"fmt"
	"html"
	"remotehunter/internal/models"
	"strings"
)

// BuildATSTemplateHTML renders a structured resume into an elegant Times New Roman HTML/CSS single-page template.
func BuildATSTemplateHTML(sr *models.StructuredResume, fitSinglePage ...bool) string {
	if sr == nil {
		return ""
	}

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
            margin: 12px 18px;
        }
        body {
            font-family: "Times New Roman", Times, serif;
            color: #000;
            line-height: 1.2;
            font-size: 13px;
            padding: 10px 15px;
            max-width: 850px;
            margin: 0 auto;
            background-color: #fff;
        }
        
        header {
            text-align: center;
            margin-bottom: 8px;
        }
        h1 {
            font-family: "Times New Roman", Times, serif;
            font-size: 24px;
            font-weight: bold;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin: 0 0 3px 0;
            text-align: center;
        }
        .subtitle {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: 14px;
            margin: 0 0 4px 0;
            text-align: center;
        }
        .contact-info {
            font-family: "Times New Roman", Times, serif;
            font-size: 12px;
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

        h2 {
            font-family: "Times New Roman", Times, serif;
            font-size: 14.5px;
            margin-top: 8px;
            margin-bottom: 5px;
            text-transform: uppercase;
            border-bottom: 1.5px solid #000;
            border-top: none;
            padding-bottom: 2px;
            font-weight: bold;
        }

        p {
            font-family: "Times New Roman", Times, serif;
            margin: 0 0 8px 0;
            font-size: 13px;
            text-align: justify;
        }
        
        .flex-between {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
        }

        .job-title-container {
            margin-bottom: 1px;
            font-size: 13.5px;
        }
        .job-title {
            font-family: "Times New Roman", Times, serif;
            font-size: 13.5px;
        }
        .job-title strong {
            font-weight: bold;
        }
        .job-date {
            font-family: "Times New Roman", Times, serif;
            font-size: 13px;
        }
        .company-container {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: 13px;
            margin-bottom: 3px;
        }
        .company-name, .job-location {
            font-style: italic;
        }

        ul {
            font-family: "Times New Roman", Times, serif;
            margin: 0 0 4px 0;
            padding-left: 20px;
            font-size: 13px;
            text-align: justify;
            list-style-type: disc !important;
        }
        li {
            font-family: "Times New Roman", Times, serif;
            margin-bottom: 2px;
            line-height: 1.2;
            list-style-type: disc !important;
        }
        li strong {
            font-weight: bold;
        }

        .tech-used {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: 12.5px;
            margin-top: 3px;
            margin-bottom: 6px;
        }

        .edu-details {
            font-family: "Times New Roman", Times, serif;
            font-style: italic;
            font-size: 13px;
            margin-top: 1px;
            margin-bottom: 5px;
        }

        .skills-table {
            font-family: "Times New Roman", Times, serif;
            width: 100%;
            font-size: 13px;
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
		} else if strings.Contains(cleanItem, "+") || strings.Contains(cleanItem, "-") || len(cleanItem) >= 10 {
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

	if len(sr.WorkExperience) > 0 {
		sb.WriteString(`

    <section>
        <h2>WORK EXPERIENCE</h2>`)
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
		sb.WriteString(`

    <section>
        <h2>EDUCATION</h2>`)
		for _, edu := range sr.Education {
			sb.WriteString(fmt.Sprintf(`
        <div class="flex-between" style="font-size: 13.5px; font-family: 'Times New Roman', Times, serif;">
            <div><strong>%s</strong></div>
            <div>%s</div>
        </div>
        <div class="edu-details"><em>%s</em></div>`,
				renderFormattedTextGo(edu.Institution),
				html.EscapeString(edu.Date),
				renderFormattedTextGo(edu.Degree),
			))
		}
		sb.WriteString(`
    </section>`)
	}

	sb.WriteString(`

</body>
</html>`)

	return sb.String()
}
