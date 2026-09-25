package scraper

import (
	"testing"
	"time"
)

func TestParseRelativeDate(t *testing.T) {
	now := time.Now()

	cases := []struct {
		in       string
		wantDays int
		wantNil  bool
	}{
		// long form (Bayt, ZipRecruiter, Naukri-style pages)
		{"3 days ago", 3, false},
		{"2+ weeks ago", 14, false},
		{"1 month ago", 30, false},
		{"Today", 0, false},
		{"Just now", 0, false},
		{"Yesterday", 1, false},
		{"2 hours ago", 0, false},
		// compact form (Glassdoor)
		{"30d+", 30, false},
		{"9d", 9, false},
		{"24h", 0, false},
		{"3w", 21, false},
		{"6mo", 180, false},
		{"1y", 365, false},
		// unrecognised
		{"", 0, true},
		{"Remote", 0, true},
	}

	for _, c := range cases {
		got := parseRelativeDate(c.in)
		if c.wantNil {
			if got != nil {
				t.Errorf("parseRelativeDate(%q) = %v, want nil", c.in, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("parseRelativeDate(%q) = nil, want ~%d days ago", c.in, c.wantDays)
			continue
		}
		// Tolerate ±14 days: calendar months/years vary in length and
		// same-day timestamps truncate. "hours/minutes ago" land on day 0.
		gotDays := now.Sub(*got).Hours() / 24
		if diff := gotDays - float64(c.wantDays); diff > 14 || diff < -14 {
			t.Errorf("parseRelativeDate(%q) = %.1f days ago, want ~%d (parsed %v)", c.in, gotDays, c.wantDays, got)
		}
	}
}
