package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseBlankLines(t *testing.T) {
	tests := []struct {
		name     string
		original string
		output   string
		expected string
	}{
		{
			name:     "leftover lines at the top are removed",
			original: "resource \"a\" \"x\" {\n}\n\nresource \"a\" \"y\" {\n}\n",
			output:   "\n\n\nresource \"b\" \"x\" {\n}\n\nresource \"b\" \"y\" {\n}\n",
			expected: "resource \"b\" \"x\" {\n}\n\nresource \"b\" \"y\" {\n}\n",
		},
		{
			name:     "leading blank lines the user wrote are kept",
			original: "\nresource \"a\" \"x\" {\n}\n",
			output:   "\n\n\nresource \"b\" \"x\" {\n}\n",
			expected: "\nresource \"b\" \"x\" {\n}\n",
		},
		{
			name:     "a run after a comment shrinks to the original's longest",
			original: "# records\n\nresource \"a\" \"x\" {\n}\n",
			output:   "# records\n\n\n\nresource \"b\" \"x\" {\n}\n",
			expected: "# records\n\nresource \"b\" \"x\" {\n}\n",
		},
		{
			name:     "runs as long as the original's are kept (heredocs, spacing)",
			original: "x = <<EOT\na\n\n\nb\nEOT\n",
			output:   "x = <<EOT\na\n\n\nb\nEOT\n\n\n\n\nresource \"b\" \"x\" {\n}\n",
			expected: "x = <<EOT\na\n\n\nb\nEOT\n\n\nresource \"b\" \"x\" {\n}\n",
		},
		{
			name:     "no blank lines in the original still allows one between blocks",
			original: "resource \"a\" \"x\" {\n}\n",
			output:   "resource \"b\" \"x\" {\n}\n\nmoved {\n}\n",
			expected: "resource \"b\" \"x\" {\n}\n\nmoved {\n}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collapseBlankLines([]byte(tt.output), measureBlankLines([]byte(tt.original)))
			assert.Equal(t, tt.expected, string(got))
		})
	}
}
