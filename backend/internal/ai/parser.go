package ai

import (
	"html"
	"regexp"
	"strings"
)

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// stripHTMLForPrompt removes HTML tags, CSS styles, and SVG markup to optimize prompt token size
func stripHTMLForPrompt(htmlStr string) string {
	if !strings.Contains(htmlStr, "<") {
		return htmlStr
	}
	// Remove <style>...</style> content completely
	reStyle := regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	s := reStyle.ReplaceAllString(htmlStr, "")

	// Remove <script>...</script> content completely
	reScript := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	s = reScript.ReplaceAllString(s, "")

	// Remove <head>...</head> content completely
	reHead := regexp.MustCompile(`(?is)<head.*?>.*?</head>`)
	s = reHead.ReplaceAllString(s, "")

	// Remove <svg>...</svg> content completely
	reSvg := regexp.MustCompile(`(?is)<svg.*?>.*?</svg>`)
	s = reSvg.ReplaceAllString(s, "")

	// Replace block elements with newlines
	reBlock := regexp.MustCompile(`(?i)</?(?:p|div|li|tr|h[1-6]|header|section|br)\b[^>]*>`)
	s = reBlock.ReplaceAllString(s, "\n")

	// Strip remaining HTML tags
	reTag := regexp.MustCompile(`<[^>]*>`)
	s = reTag.ReplaceAllString(s, "")

	// Unescape HTML entities
	s = html.UnescapeString(s)

	// Clean up multiple newlines
	reLines := regexp.MustCompile(`\n{3,}`)
	s = reLines.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

func renderFormattedTextGo(str string) string {
	s := regexp.MustCompile(`^[•\-▪◦\s]+`).ReplaceAllString(str, "")
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
	s := regexp.MustCompile(`^[•\-▪◦\s]+`).ReplaceAllString(str, "")
	s = html.EscapeString(strings.TrimSpace(s))

	s = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(s, "<strong>$1</strong>")
	s = regexp.MustCompile(`(?i)&lt;strong&gt;(.*?)&lt;/strong&gt;`).ReplaceAllString(s, "<strong>$1</strong>")
	s = regexp.MustCompile(`(?i)&lt;b&gt;(.*?)&lt;/b&gt;`).ReplaceAllString(s, "<strong>$1</strong>")

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
