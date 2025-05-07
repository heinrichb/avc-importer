package utils

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout during the execution of f() and returns the captured output.
func captureStdout(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestPrintColored exercises all branches of the PrintColored function in a table-driven test.
func TestPrintColored(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
		// expectedContains lists substrings that must appear in the output.
		expectedContains []string
		// expectEmpty indicates that no output should be produced.
		expectEmpty bool
	}{
		{
			name:        "No arguments: nothing should be printed",
			args:        []interface{}{},
			expectEmpty: true,
		},
		{
			name:        "Non-string first argument: invalid input produces no output",
			args:        []interface{}{123},
			expectEmpty: true,
		},
		{
			name:             "Single string: prints in default white",
			args:             []interface{}{"Just a test string"},
			expectedContains: []string{"Just a test string"},
		},
		{
			name:             "Two strings with hex color",
			args:             []interface{}{"Prefix: ", "Value", "#FF5733"},
			expectedContains: []string{"\033[38;2;255;87;51mPrefix: \033[0mValue"},
		},
		{
			name:             "Two strings with invalid hex: defaults to white",
			args:             []interface{}{"Prefix: ", "Value", "notAColor"},
			expectedContains: []string{"\033[0mPrefix: \033[0mValue"},
		},
	}

	// Iterate over each test case.
	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			output := captureStdout(func() {
				PrintColored(tc.args...)
			})
			if tc.expectEmpty {
				if output != "" {
					t.Errorf("Expected no output, but got: %q", output)
				}
				return
			}
			// Verify that each expected substring is present in the output.
			for _, substr := range tc.expectedContains {
				if !strings.Contains(output, substr) {
					t.Errorf("Test case %q: expected output to contain %q, but it did not. Full output: %q", tc.name, substr, output)
				}
			}
		})
	}
}
