package scraper

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncateStrKeepsValidUTF8 guards against the byte-slicing bug that split
// multi-byte runes and produced invalid UTF-8 for non-Latin job titles.
func TestTruncateStrKeepsValidUTF8(t *testing.T) {
	// Each of these is 3 bytes; slicing at byte 250 must not split a rune.
	multibyte := strings.Repeat("日", 200) // 600 bytes, 200 runes

	if got := truncateStr(multibyte, 250); got != multibyte {
		t.Errorf("200 runes must not be truncated to 250 runes: got %d runes", utf8.RuneCountInString(got))
	}

	got := truncateStr(multibyte, 10)
	if !utf8.ValidString(got) {
		t.Errorf("truncateStr produced invalid UTF-8: %q", got)
	}
	if n := utf8.RuneCountInString(got); n != 10 {
		t.Errorf("want 10 runes, got %d", n)
	}

	if got := truncateStr("short", 100); got != "short" {
		t.Errorf("string under limit must be unchanged, got %q", got)
	}
}

// TestSanitizeFieldStripsNUL covers the NUL bytes that PostgreSQL text columns
// reject outright.
func TestSanitizeFieldStripsNUL(t *testing.T) {
	if got := sanitizeField("a\x00b"); got != "ab" {
		t.Errorf("want NUL stripped, got %q", got)
	}
	if got := sanitizeField("clean"); got != "clean" {
		t.Errorf("clean input must be unchanged, got %q", got)
	}
	if !utf8.ValidString(sanitizeField("bad\xff\xfeutf8")) {
		t.Errorf("invalid UTF-8 must be replaced")
	}
}
