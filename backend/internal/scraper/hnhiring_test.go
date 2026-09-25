package scraper

import "testing"

// TestParseHNHeader pins the pipe-delimited header parse. The old scraper split
// the whole comment on "|" and took element 0 as the company, so postings whose
// first line was a location ("Location: London, UK") produced a company of
// "Location: London, UK" and a title of the next line's fragment.
//
// Headers arrive as raw Algolia HTML: paragraph breaks are <p>, not newlines.
func TestParseHNHeader(t *testing.T) {
	cases := []struct {
		name        string
		text        string
		wantCompany string
		wantTitle   string
		wantLoc     string
	}{
		{
			name:        "standard pipe header",
			text:        "NetBird | Senior Go Engineer | Berlin, Germany (or remote) | Full-time<p>We build WireGuard overlays.",
			wantCompany: "NetBird",
			wantTitle:   "Senior Go Engineer",
			wantLoc:     "Berlin, Germany",
		},
		{
			name:        "company with trailing url",
			text:        "Tether (https://tether.io/) | Backend Engineer | Remote | Full-time<p>About us.",
			wantCompany: "Tether",
			wantTitle:   "Backend Engineer",
			wantLoc:     "Remote",
		},
		{
			name:        "no pipe means no guess",
			text:        "We are hiring a Go engineer to work remotely on distributed systems.",
			wantCompany: "",
			wantTitle:   "",
			wantLoc:     "",
		},
		{
			name:        "candidate resume post is rejected",
			text:        "Location: Jakarta, Indonesia (UTC+7) Remote: Yes Willing to relocate: Yes<p>Technologies: Java Springboot, Go",
			wantCompany: "",
			wantTitle:   "",
			wantLoc:     "",
		},
		{
			name:        "prose first line is rejected",
			text:        "Location: London, UK | Remote: Yes<p>I am a backend engineer.",
			wantCompany: "",
			wantTitle:   "",
			wantLoc:     "",
		},
		{
			name:        "location in role cell is rejected",
			text:        "NetBird | Berlin, Germany | Full-time<p>NetBird is open source networking.",
			wantCompany: "",
			wantTitle:   "",
			wantLoc:     "",
		},
		{
			name:        "role-less placeholder is rejected",
			text:        "Sumble | Multiple Roles | Remote<p>We are hiring.",
			wantCompany: "",
			wantTitle:   "",
			wantLoc:     "",
		},
		{
			name:        "leading blank line is skipped",
			text:        "<p>Astronomer | Software Engineer | NYC<p>We do data orchestration.",
			wantCompany: "Astronomer",
			wantTitle:   "Software Engineer",
			wantLoc:     "NYC",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotCompany, gotTitle, gotLoc := parseHNHeader(tc.text)
			if gotCompany != tc.wantCompany || gotTitle != tc.wantTitle || gotLoc != tc.wantLoc {
				t.Errorf("parseHNHeader() = (%q, %q, %q), want (%q, %q, %q)",
					gotCompany, gotTitle, gotLoc, tc.wantCompany, tc.wantTitle, tc.wantLoc)
			}
		})
	}
}

// TestGoKeywordMatching guards the keyword filter against substring hits:
// "Google", "Pango" and "category" all contain "go".
func TestGoKeywordMatching(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"We use Go and Postgres", true},
		{"Golang experience required", true},
		{"Senior Engineer at Pango", false},
		{"Join Google", false},
		{"We are a category leader", false},
		{"goroutine scheduler", false},
	}

	for _, tc := range cases {
		if got := goKeyword.MatchString(tc.text); got != tc.want {
			t.Errorf("goKeyword.MatchString(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

