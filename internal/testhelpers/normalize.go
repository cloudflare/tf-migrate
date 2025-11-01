package testhelpers

import (
	"regexp"
	"strings"
)

// NormalizeHCLWhitespace normalizes whitespace in HCL output for test comparisons.
// This handles cases where block removal leaves behind extra newlines.
func NormalizeHCLWhitespace(content string) string {
	// Normalize multiple consecutive newlines to at most 2
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	
	// Remove blank lines between the last attribute and first block
	// This pattern matches: attribute = value, then 2+ newlines, then a block
	content = regexp.MustCompile(`(\n\s*\w+\s*=\s*[^\n]+)\n\n+(\s*\w+\s*{)`).ReplaceAllString(content, "$1\n$2")
	
	// Trim trailing whitespace on each line
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	
	return strings.Join(lines, "\n")
}