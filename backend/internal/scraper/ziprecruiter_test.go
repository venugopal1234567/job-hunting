package scraper

import (
	"testing"
)

// TestInferCountryZipRecruiter pins the region-code matching. A bare
// strings.Contains(loc, "ca") used to label Chicago, Cary and every "City, CA"
// posting as Canada, which kept US jobs in the feed and dropped the rest.
func TestInferCountryZipRecruiter(t *testing.T) {
	cases := []struct {
		location string
		want     string
	}{
		{"Fremont, CA", "US"},
		{"San Francisco, CA", "US"},
		{"Sunnyvale, CA", "US"},
		{"Chicago, IL", "US"},
		{"Cary, NC", "US"},
		{"New York, NY", "US"},
		{"Toronto, ON", "Canada"},
		{"Vancouver, BC", "Canada"},
		{"Canada", "Canada"},
		{"Bengaluru, India", "India"},
		{"Pune", "India"},
		{"Remote", "Worldwide"},
		{"", "Worldwide"},
		{"Worldwide", "Worldwide"},
	}

	for _, tc := range cases {
		if got := inferCountryZipRecruiter(tc.location); got != tc.want {
			t.Errorf("inferCountryZipRecruiter(%q) = %q, want %q", tc.location, got, tc.want)
		}
	}
}

// TestHasTrailingRegionCode covers the boundary the substring match got wrong.
func TestHasTrailingRegionCode(t *testing.T) {
	cases := []struct {
		loc   string
		codes string
		want  bool
	}{
		{"fremont, ca", usStateCodes, true},
		{"cary, nc", usStateCodes, true},
		{"toronto, on", canadaProvinceCodes, true},
		{"chicago", usStateCodes, false}, // no region code, just the substring "ca"
		{"cary", usStateCodes, false},
		{"canada", usStateCodes, false},
		{"remote", usStateCodes, false},
		{"", usStateCodes, false},
		{"ca", usStateCodes, false}, // too short to carry a separator
	}

	for _, tc := range cases {
		if got := hasTrailingRegionCode(tc.loc, tc.codes); got != tc.want {
			t.Errorf("hasTrailingRegionCode(%q) = %v, want %v", tc.loc, got, tc.want)
		}
	}
}

// TestIsGoRole guards the arbeitnow filter: it must accept Go titles and reject
// titles that merely contain the letters "go".
func TestIsGoRole(t *testing.T) {
	accept := []string{
		"Senior Golang Engineer",
		"Backend Engineer (Go)",
		"Go Developer",
		"Staff Software Engineer, Go",
	}
	reject := []string{
		"Going Concern Accountant",
		"Mongolian Translator",
		"Go-to-Market Manager",
		"Python Developer",
	}

	for _, title := range accept {
		if !isGoRole(title) {
			t.Errorf("isGoRole(%q) = false, want true", title)
		}
	}
	for _, title := range reject {
		if isGoRole(title) {
			t.Errorf("isGoRole(%q) = true, want false", title)
		}
	}
}
