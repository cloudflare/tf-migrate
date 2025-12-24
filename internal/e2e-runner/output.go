// output.go provides colorized terminal output utilities for test reporting.
//
// This file implements a consistent set of functions for printing colored
// messages to the terminal, making test output more readable and helping
// users quickly identify successes, warnings, and errors. All output
// functions use ANSI color codes for terminal compatibility.
package e2e

import (
	"fmt"
)

const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[1;34m" // Bright blue for better visibility
	colorCyan   = "\033[0;36m"
	colorReset  = "\033[0m"
)

func printRed(format string, args ...interface{}) {
	fmt.Printf(colorRed+format+colorReset+"\n", args...)
}

func printGreen(format string, args ...interface{}) {
	fmt.Printf(colorGreen+format+colorReset+"\n", args...)
}

func printYellow(format string, args ...interface{}) {
	fmt.Printf(colorYellow+format+colorReset+"\n", args...)
}

func printBlue(format string, args ...interface{}) {
	fmt.Printf(colorBlue+format+colorReset+"\n", args...)
}

func printCyan(format string, args ...interface{}) {
	fmt.Printf(colorCyan+format+colorReset+"\n", args...)
}

func printSuccess(format string, args ...interface{}) {
	if len(args) > 0 {
		printGreen("✓ "+format, args...)
	} else {
		printGreen("✓ " + format)
	}
}

func printError(format string, args ...interface{}) {
	if len(args) > 0 {
		printRed("✗ "+format, args...)
	} else {
		printRed("✗ " + format)
	}
}

func printHeader(title string) {
	fmt.Println()
	printCyan("========================================")
	printCyan(title)
	printCyan("========================================")
	fmt.Println()
}
