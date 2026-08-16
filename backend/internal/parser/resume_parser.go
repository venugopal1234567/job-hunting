// Package parser provides a modular, regex-safe fallback parser that converts
// plain-text resume content into a models.StructuredResume.
//
// Design goals:
//   - All compiled regexes live as package-level variables so they are compiled
//     exactly once at program startup, not on every call.
//   - Each parsing stage (header, contact, section body) is a discrete
//     function, making each piece independently testable.
//   - No hard-coded personal data in fallback skill lists – instead an
//     empty but valid slice is returned when no skills are detected.
package parser

import (
	"regexp"
	"remotehunter/internal/models"
	"strings"
)

// ── Compiled regexes (initialized once at startup) ────────────────────────────

var (
	// reCite strips citation markers injected by AI responses.
	reCite = regexp.MustCompile(`\[cite:\s*\d+\]`)

	// reEmail matches a standard e-mail address.
	reEmail = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

	// rePhone matches international and local phone number formats.
	rePhone = regexp.MustCompile(`(?:\+?\d{1,3}[\s\-]?)?\(?\d{3,5}\)?[\s\-]?\d{3,5}[\s\-]?\d{3,5}`)

	// reJobTitle matches common professional title suffixes following optional
	// prefixes such as "Senior", "Lead", "Staff", etc.
	reJobTitle = regexp.MustCompile(`(?i)^([A-Za-z\s,\(\)\/\-]+?(?:Engineer|Developer|Architect|Manager|Lead|Specialist|Consultant|Scientist|Designer|Analyst|Programmer))`)

	// reDateRange matches a full date range like "Jan 2020 – Present" or
	// "2018 - 2021".
	reDateRange = regexp.MustCompile(`(?i)((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4})[-–\s\x{2013}\x{2014}]+(?:Present|\d{4}|Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec))`)

	// reTrailingDate matches a trailing single month/year token used to stitch
	// together two-line date ranges.
	reTrailingDate = regexp.MustCompile(`(?i)((?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec|\d{4}))\s*$`)

	// reSeparator matches common separator characters used inside job lines.
	reSeparator = regexp.MustCompile(`[|–—\-]+`)

	// knownSectionHeaders is the canonical set of uppercase section names
	// used as sentinels in the state machine.
	knownSectionHeaders = []string{
		"PROFESSIONAL SUMMARY", "SUMMARY",
		"WORK EXPERIENCES", "WORK EXPERIENCE", "EXPERIENCE",
		"SKILLS", "TECHNICAL SKILLS",
		"EDUCATIONS", "EDUCATION",
		"PROJECTS", "CERTIFICATIONS",
	}

	// bulletPrefixes is the set of characters used to introduce bullet points.
	bulletPrefixCutset = "•-*▪◦ "
)

// ── Public API ────────────────────────────────────────────────────────────────

// ParseText converts raw plain-text resume content into a StructuredResume.
// It is safe to call concurrently because all state is local to the function.
func ParseText(text string) *models.StructuredResume {
	// Strip citation markers left by AI systems.
	text = reCite.ReplaceAllString(text, "")

	lines := nonEmptyLines(text)
	res := &models.StructuredResume{
		ContactItems:   []string{},
		WorkExperience: []models.JobExperience{},
		Education:      []models.EducationItem{},
		Skills:         []models.SkillCategory{},
	}
	if len(lines) == 0 {
		return res
	}

	idx := parseHeader(lines, res)
	parseBody(lines[idx:], res)
	finalizeSkills(res)

	return res
}

// IsHTMLContent reports whether text already looks like an HTML document so
// the caller can skip parsing and use the string directly.
func IsHTMLContent(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 {
		return false
	}
	// Check first character for '<' or look for the word "html" in the first
	// 50 characters (without slicing past the end of the string).
	prefix := trimmed
	if len(prefix) > 50 {
		prefix = prefix[:50]
	}
	return strings.HasPrefix(trimmed, "<") || strings.Contains(strings.ToLower(prefix), "html")
}

// ── Header parsing ────────────────────────────────────────────────────────────

// parseHeader extracts the candidate's name, title, and contact items from the
// top of the resume (lines before the first recognised section header).
// It returns the index of the first line that was NOT consumed.
func parseHeader(lines []string, res *models.StructuredResume) int {
	if len(lines) == 0 {
		return 0
	}

	// First non-empty line is always the candidate name.
	res.Name = lines[0]
	idx := 1

	for idx < len(lines) {
		line := lines[idx]
		if isSectionHeader(line) {
			break
		}

		hasEmail := reEmail.MatchString(line)
		hasPhone := rePhone.MatchString(line)

		if res.Title == "" && !hasEmail && !hasPhone && !strings.Contains(line, "|") && len(line) < 80 {
			// Short, clean line with no contact markers → professional title.
			res.Title = line
		} else {
			// Harvest contact tokens from this line.
			workLine := line

			// Try inline title extraction when title is still missing.
			if res.Title == "" {
				if m := reJobTitle.FindStringSubmatch(workLine); len(m) > 1 {
					res.Title = strings.TrimSpace(m[1])
					workLine = strings.TrimSpace(workLine[len(m[0]):])
				}
			}

			if em := reEmail.FindString(workLine); em != "" {
				appendUnique(&res.ContactItems, em)
				workLine = strings.TrimSpace(strings.Replace(workLine, em, " ", 1))
			}
			if pm := rePhone.FindString(workLine); pm != "" {
				appendUnique(&res.ContactItems, pm)
				workLine = strings.TrimSpace(strings.Replace(workLine, pm, " ", 1))
			}
			for _, part := range strings.Split(workLine, "|") {
				if p := strings.TrimSpace(part); p != "" {
					appendUnique(&res.ContactItems, p)
				}
			}
		}
		idx++
	}
	return idx
}

// ── Body parsing (section state machine) ─────────────────────────────────────

type section int

const (
	sectionUnknown section = iota
	sectionSummary
	sectionSkills
	sectionWork
	sectionEducation
)

// parseBody drives the section-level state machine over the remaining lines.
func parseBody(lines []string, res *models.StructuredResume) {
	current := sectionUnknown
	var currentJob *models.JobExperience
	var currentEdu *models.EducationItem

	flush := func() {
		if currentJob != nil {
			res.WorkExperience = append(res.WorkExperience, *currentJob)
			currentJob = nil
		}
		if currentEdu != nil {
			res.Education = append(res.Education, *currentEdu)
			currentEdu = nil
		}
	}

	for _, line := range lines {
		if isSectionHeader(line) {
			flush()
			current = classifySection(line)
			continue
		}

		switch current {
		case sectionSummary:
			parseSummaryLine(line, res)
		case sectionSkills:
			parseSkillLine(line, res)
		case sectionWork:
			parseWorkLine(line, res, &currentJob)
		case sectionEducation:
			parseEducationLine(line, res, &currentEdu)
		}
	}
	flush()
}

// ── Section classifiers ───────────────────────────────────────────────────────

func classifySection(line string) section {
	upper := strings.ToUpper(strings.TrimSpace(line))
	switch {
	case strings.Contains(upper, "SUMMARY"):
		return sectionSummary
	case strings.Contains(upper, "SKILL"):
		return sectionSkills
	case strings.Contains(upper, "WORK") || strings.Contains(upper, "EXPERIENCE"):
		return sectionWork
	case strings.Contains(upper, "EDU"):
		return sectionEducation
	default:
		return sectionUnknown
	}
}

// isSectionHeader returns true when line is a known or all-caps section header.
func isSectionHeader(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	for _, h := range knownSectionHeaders {
		if upper == h {
			return true
		}
	}
	// Heuristic: short, all-uppercase line with no separators → likely a header.
	return len(line) < 40 &&
		line == upper &&
		!strings.Contains(line, "|") &&
		!strings.Contains(line, "@") &&
		len(line) > 3
}

// ── Per-section line parsers ──────────────────────────────────────────────────

func parseSummaryLine(line string, res *models.StructuredResume) {
	if res.Summary == "" {
		res.Summary = line
	} else {
		res.Summary += " " + line
	}
}

// parseSkillLine adds a skill category parsed from line into res.
func parseSkillLine(line string, res *models.StructuredResume) {
	if colon := strings.Index(line, ":"); colon >= 0 {
		cat := strings.TrimSpace(line[:colon])
		items := strings.TrimSpace(line[colon+1:])
		res.Skills = append(res.Skills, models.SkillCategory{Category: cat, Items: items})
		return
	}
	txt := strings.TrimLeft(line, bulletPrefixCutset)
	if txt == "" {
		return
	}
	// Append to the last category when its Items field is still empty.
	if n := len(res.Skills); n > 0 && res.Skills[n-1].Items == "" {
		res.Skills[n-1].Items = txt
		return
	}
	res.Skills = append(res.Skills, models.SkillCategory{Category: "Key Skills", Items: txt})
}

func parseWorkLine(line string, res *models.StructuredResume, cur **models.JobExperience) {
	// Technology stack line.
	if strings.HasPrefix(strings.ToLower(line), "technologies") {
		if *cur != nil {
			if i := strings.Index(line, ":"); i >= 0 {
				(*cur).TechStack = strings.TrimSpace(line[i+1:])
			} else {
				(*cur).TechStack = line
			}
		}
		return
	}

	fullMatch := reDateRange.FindStringSubmatch(line)
	trailMatch := reTrailingDate.FindStringSubmatch(line)

	// Stitch together a two-line date range: first line ended with a start
	// date, this line begins with the end date.
	if *cur != nil && (*cur).Date != "" &&
		!strings.Contains((*cur).Date, "-") &&
		!strings.Contains((*cur).Date, "Present") &&
		len(trailMatch) > 1 {

		endDate := trailMatch[1]
		(*cur).Date += " - " + endDate
		comp := strings.TrimSpace(reTrailingDate.ReplaceAllString(line, ""))
		if comp != "" {
			(*cur).Company = comp
		}
		return
	}

	// New job with a full date range on the same line.
	if len(fullMatch) > 0 && len(line) < 130 && !strings.HasPrefix(line, "•") {
		if *cur != nil {
			res.WorkExperience = append(res.WorkExperience, **cur)
		}
		*cur = &models.JobExperience{
			Title:   strings.TrimSpace(reSeparator.ReplaceAllString(reDateRange.ReplaceAllString(line, ""), "")),
			Date:    fullMatch[1],
			Bullets: []string{},
		}
		return
	}

	// New job with only a trailing date token.
	if len(trailMatch) > 0 && len(line) < 100 &&
		(*cur == nil || len((*cur).Bullets) == 0) &&
		!strings.HasPrefix(line, "•") {

		if *cur != nil {
			res.WorkExperience = append(res.WorkExperience, **cur)
		}
		*cur = &models.JobExperience{
			Title:   strings.TrimSpace(reSeparator.ReplaceAllString(reTrailingDate.ReplaceAllString(line, ""), "")),
			Date:    trailMatch[1],
			Bullets: []string{},
		}
		return
	}

	// Company / location line under an open job entry.
	if *cur != nil && (*cur).Company == "" &&
		(strings.Contains(line, "|") ||
			strings.Contains(line, " - ") ||
			strings.Contains(line, "–") ||
			strings.Contains(strings.ToLower(line), "full time") ||
			strings.Contains(strings.ToLower(line), "remote")) {

		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '|' || r == '-' || r == '–'
		})
		if len(parts) >= 2 {
			(*cur).Company = strings.TrimSpace(parts[0])
			(*cur).Location = strings.TrimSpace(parts[1])
		} else {
			(*cur).Location = line
		}
		return
	}

	// Supplementary location line.
	if *cur != nil && (*cur).Location == "" &&
		(strings.Contains(strings.ToLower(line), "full time") ||
			strings.Contains(strings.ToLower(line), "remote") ||
			strings.Contains(line, "India")) {
		(*cur).Location = line
		return
	}

	// Bullet point under open job.
	if *cur != nil {
		if txt := strings.TrimLeft(line, bulletPrefixCutset); txt != "" {
			(*cur).Bullets = append((*cur).Bullets, txt)
		}
		return
	}

	// Orphaned line with no open job — start a new job with title only.
	*cur = &models.JobExperience{Title: line, Bullets: []string{}}
}

func parseEducationLine(line string, res *models.StructuredResume, cur **models.EducationItem) {
	dm := reDateRange.FindStringSubmatch(line)

	// Attach date to an open education entry.
	if *cur != nil && (*cur).Date == "" && len(dm) > 0 {
		(*cur).Date = dm[1]
		if deg := strings.TrimSpace(reDateRange.ReplaceAllString(line, "")); deg != "" {
			(*cur).Degree = deg
		}
		return
	}

	// New education entry.
	if len(dm) > 0 {
		if *cur != nil {
			res.Education = append(res.Education, **cur)
		}
		*cur = &models.EducationItem{
			Institution: strings.TrimSpace(reSeparator.ReplaceAllString(reDateRange.ReplaceAllString(line, ""), "")),
			Date:        dm[1],
		}
		return
	}

	// Degree detail under open education entry.
	if *cur != nil {
		if (*cur).Degree == "" {
			(*cur).Degree = line
		} else {
			(*cur).Degree += " " + line
		}
		return
	}

	// Institution line with no date yet.
	*cur = &models.EducationItem{Institution: line}
}

// ── Post-processing ───────────────────────────────────────────────────────────

// finalizeSkills removes skill categories with empty Items and, when no skills
// were detected at all, leaves the slice empty so the caller can decide how to
// handle the gap (rather than injecting hard-coded personal data).
func finalizeSkills(res *models.StructuredResume) {
	var valid []models.SkillCategory
	for _, s := range res.Skills {
		if strings.TrimSpace(s.Items) != "" {
			valid = append(valid, s)
		}
	}
	if valid == nil {
		valid = []models.SkillCategory{}
	}
	res.Skills = valid
}

// ── Utilities ─────────────────────────────────────────────────────────────────

// nonEmptyLines splits text on newlines and discards blank lines.
func nonEmptyLines(text string) []string {
	raw := strings.Split(text, "\n")
	out := make([]string, 0, len(raw))
	for _, l := range raw {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// appendUnique appends s to *slice only when s is not already present.
func appendUnique(slice *[]string, s string) {
	for _, item := range *slice {
		if item == s {
			return
		}
	}
	*slice = append(*slice, s)
}
