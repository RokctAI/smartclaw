package agent

import (
	"testing"
)

func TestMinifySystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "removes consecutive blank lines",
			input: "line 1\n\n\n\nline 2",
			expected: "line 1\n\nline 2",
		},
		{
			name: "trims leading and trailing spaces",
			input: "  line 1  \n\n  line 2  ",
			expected: "line 1\n\nline 2",
		},
		{
			name: "handles multiple blank lines with spaces",
			input: "line 1\n  \n \n \nline 2",
			expected: "line 1\n\nline 2",
		},
		{
			name: "trims empty start and end",
			input: "\n\n  \nline 1\nline 2\n\n ",
			expected: "line 1\nline 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minifySystemPrompt(tt.input)
			if result != tt.expected {
				t.Errorf("minifySystemPrompt() = %q, want %q", result, tt.expected)
			}
		})
	}
}
