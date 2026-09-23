package scraper

import (
	"testing"
	"time"
)

// TestParseRemoteRocketshipDate pins the three label shapes the Remote Rocketship
// cards actually render. The relative form used to fall through to nil, which the
// jobs query reads as "posted within the window" — a 6-day-old posting surfaced
// under the 24-hour filter.
func TestParseRemoteRocketshipDate(t *testing.T) {
	now := time.Now()

	daysAgo := func(d int) string {
		return now.AddDate(0, 0, -d).Format("2006-01-02")
	}

	cases := []struct {
		label       string
		wantNil     bool
		wantDate    string // "2006-01-02", checked only when non-empty
		description string
	}{
		{label: "6 days ago", description: "relative day label must parse", wantDate: daysAgo(6)},
		{label: "2 weeks ago", description: "relative week label must parse", wantDate: daysAgo(14)},
		{label: "1 month ago", description: "relative month label must parse", wantDate: now.AddDate(0, -1, 0).Format("2006-01-02")},
		{label: "3 hours ago", description: "relative hour label must parse"},
		{label: "yesterday", description: "yesterday must parse", wantDate: daysAgo(1)},
		{label: "today", description: "today must parse", wantDate: now.Format("2006-01-02")},
		{label: "September 15", description: "yearless month/day must parse"},
		{label: "Sep 15, 2026", description: "month/day with year must parse", wantDate: "2026-09-15"},
		{label: "2026-09-15", description: "ISO date must parse", wantDate: "2026-09-15"},
		{label: "not a date", description: "unparseable stays nil", wantNil: true},
		{label: "", description: "empty stays nil", wantNil: true},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			got := parseRemoteRocketshipDate(tc.label)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("label %q: got %v, want nil", tc.label, got.Format(time.RFC3339))
				}
				return
			}
			if got == nil {
				t.Fatalf("label %q: got nil, want a date", tc.label)
			}
			if tc.wantDate != "" && got.Format("2006-01-02") != tc.wantDate {
				t.Errorf("label %q: got %s, want %s", tc.label, got.Format("2006-01-02"), tc.wantDate)
			}
			// A future posting date would also fall outside every window.
			if got.After(now.Add(time.Minute)) {
				t.Errorf("label %q: parsed date %s is in the future", tc.label, got.Format(time.RFC3339))
			}
		})
	}
}

// TestParseRemoteRocketshipDateYearlessRollsBack covers the year-inference branch:
// a yearless label read in January for a December date is last year, not this one.
func TestParseRemoteRocketshipDateYearlessRollsBack(t *testing.T) {
	got := parseRemoteRocketshipDate("January 2")
	if got == nil {
		t.Fatal("got nil, want a date")
	}
	if got.After(time.Now()) {
		t.Errorf("got %s, which is in the future", got.Format(time.RFC3339))
	}
}
