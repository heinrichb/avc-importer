package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/heinrichb/awsimporter/pkg/utils"
)

// detailedCoverageRegex matches typical coverage detail lines from `go tool cover -func`.
// Example:
//
//	github.com/.../file.go:31:     funcName                100.0%
var detailedCoverageRegex = regexp.MustCompile(`^([^:]+\.go):(\d+):(\s+)(\S+)(\s+)([0-9]+\.[0-9]+%)$`)

// fallbackCoverageRegex matches coverage percentages in fallback lines (e.g. "total: (statements) 70.0%").
var fallbackCoverageRegex = regexp.MustCompile(`([0-9]+\.[0-9]+%)`)

// Coverage thresholds.
const (
	HighCoverageThreshold   = 80.0
	MediumCoverageThreshold = 50.0
)

// Hex color codes for styling
const (
	HexDirColor     = "#FFFFFF" // White
	HexFileColor    = "#00FFFF" // Cyan
	HexLineNumColor = "#FF00FF" // Magenta
	HexFuncColor    = "#00BFFF" // DeepSkyBlue
	HexHighCov      = "#32CD32" // LimeGreen
	HexMidCov       = "#FFFF00" // Yellow
	HexLowCov       = "#FF4500" // OrangeRed
)

// inputReader is our source for input; it defaults to os.Stdin but can be overridden in tests.
var inputReader io.Reader = os.Stdin

// exitFunc is used to exit in main(). It defaults to os.Exit but can be overridden in tests.
var exitFunc = os.Exit

// run reads from the provided reader and writes styled output to stdout.
// It returns an error if a read error occurs.
func run(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		originalLine := scanner.Text()
		styledLine := styleCoverageLine(originalLine)
		fmt.Println(styledLine)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		return err
	}
	return nil
}

// main calls run(inputReader) and uses exitFunc if an error occurs.
func main() {
	if err := run(inputReader); err != nil {
		exitFunc(1)
	}
}

// styleCoverageLine returns a styled version of the given line.
// If the line matches detailedCoverageRegex, it processes it accordingly;
// otherwise, it falls back to colorizeCoverageInLine.
func styleCoverageLine(line string) string {
	if matches := detailedCoverageRegex.FindStringSubmatch(line); matches != nil {
		fullPath := matches[1]
		lineNumber := matches[2]
		spacingBeforeFunc := matches[3]
		funcName := matches[4]
		spacingBeforeCoverage := matches[5]
		coverageString := matches[6]
		coloredFilePath := formatPathAndFile(fullPath)
		coloredLineNumber := utils.Colorize(lineNumber, HexLineNumColor)
		coloredFunction := utils.Colorize(funcName, HexFuncColor)
		coloredCoverage := colorizeCoverage(coverageString)
		return fmt.Sprintf("%s:%s:%s%s%s%s",
			coloredFilePath,
			coloredLineNumber,
			spacingBeforeFunc,
			coloredFunction,
			spacingBeforeCoverage,
			coloredCoverage,
		)
	}
	return colorizeCoverageInLine(line)
}

// formatPathAndFile splits a file path into directory and file components and colors them.
func formatPathAndFile(fullPath string) string {
	dir := filepath.Dir(fullPath)
	file := filepath.Base(fullPath)
	if dir == "." || dir == "" {
		return utils.Colorize(file, HexFileColor)
	}
	return utils.Colorize(dir+"/", HexDirColor) + utils.Colorize(file, HexFileColor)
}

// colorizeCoverageInLine replaces all coverage percentages in a line with their colored versions.
func colorizeCoverageInLine(line string) string {
	return fallbackCoverageRegex.ReplaceAllStringFunc(line, func(match string) string {
		return colorizeCoverage(match)
	})
}

// colorizeCoverage returns a colored string for the given coverage percentage.
func colorizeCoverage(coverageStr string) string {
	rawNumber := strings.TrimSuffix(coverageStr, "%")
	coverageValue, parseErr := strconv.ParseFloat(rawNumber, 64)
	if parseErr != nil {
		return coverageStr
	}
	switch {
	case coverageValue >= HighCoverageThreshold:
		return utils.Colorize(coverageStr, HexHighCov)
	case coverageValue >= MediumCoverageThreshold:
		return utils.Colorize(coverageStr, HexMidCov)
	default:
		return utils.Colorize(coverageStr, HexLowCov)
	}
}
