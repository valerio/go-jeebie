package memory

import (
	"strings"
	"unicode"
)

// cleanGameboyTitle processes a raw Game Boy ROM title by:
// 1. Converting NULL bytes to spaces
// 2. Trimming whitespace from both ends
// 3. Ensuring all characters are printable ASCII
// 4. Handling potential Japanese titles
func cleanGameboyTitle(titleBytes []byte) string {
	// Convert bytes to runes
	runes := make([]rune, 0, len(titleBytes))

	// Replace null bytes with spaces and keep only printable characters
	for _, b := range titleBytes {
		r := rune(b)
		if r == 0 {
			r = ' ' // Replace null bytes with spaces
		} else if !unicode.IsPrint(r) {
			r = '?' // Replace non-printable characters with question marks
		}
		runes = append(runes, r)
	}

	// Convert to string and trim spaces
	title := strings.TrimSpace(string(runes))

	// If title is empty after cleaning, use a placeholder
	if title == "" {
		return "(Untitled)"
	}

	return title
}
