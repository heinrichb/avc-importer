// pkg/utils/printcolor.go
package utils

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

/*
hexColorPattern matches valid 6-character hex color codes.
Example: #FFFFFF, #000000
*/
var hexColorPattern = regexp.MustCompile(`^#?([A-Fa-f0-9]{6})$`)

/*
hexToANSI converts a hex color code to an ANSI escape code for 24-bit "true color".
Example: #FF5733 → "\033[38;2;255;87;51m"
*/
func hexToANSI(hex string) string {
	if !hexColorPattern.MatchString(hex) {
		return "\033[0m" // Default color (reset)
	}

	// Remove the hash (#) if present
	hex = strings.TrimPrefix(hex, "#")

	// Parse the hex string into RGB components
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)

	// Return the ANSI escape code for true color
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

/*
FprintColored writes a colored line to the provided writer.

Parameters:
  - w: The io.Writer where output is written.
  - prefix: The string to print in the specified color (defaults to white if no color is provided).
  - secondary: The string printed immediately after the colored prefix.
  - hexColor: The color in hex string format (e.g., "#FF5733").

Usage:
	FprintColored(os.Stdout, "Loaded config from: ", configPath, "#FF5733")
*/
func FprintColored(w io.Writer, prefix, secondary, hexColor string) {
	ansiColor := hexToANSI(hexColor)
	resetColor := "\033[0m"

	if secondary != "" {
		fmt.Fprintf(w, "%s%s%s%s\n", ansiColor, prefix, resetColor, secondary)
	} else {
		fmt.Fprintln(w, ansiColor+prefix+resetColor)
	}
}

/*
PrintColored is the main exported function for this utility.
It dynamically determines how to print colored output based on the types of arguments passed.

Usage:
 1. To print a single string:
    PrintColored("Just a string")
 2. To print a prefix and secondary string with a hex color:
    PrintColored("Prefix: ", "Secondary", "#FF5733")
*/
func PrintColored(args ...interface{}) {
	if len(args) == 0 {
		return
	}

	// Otherwise, assume the first argument is a string.
	prefix, ok := args[0].(string)
	if !ok {
		return
	}

	secondary := ""
	if len(args) >= 2 {
		if sec, ok := args[1].(string); ok {
			secondary = sec
		}
	}

	hexColor := "#FFFFFF" // Default to white
	if len(args) > 2 {
		if colorStr, ok := args[2].(string); ok {
			hexColor = colorStr
		}
	}

	FprintColored(os.Stdout, prefix, secondary, hexColor)
}
