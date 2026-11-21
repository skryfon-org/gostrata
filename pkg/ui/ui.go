package ui

import (
	"fmt"
	"os"
	"strings"
)

// Color codes. They are emptied when output is not a terminal or NO_COLOR is set.
var (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
)

func init() {
	_, noColor := os.LookupEnv("NO_COLOR")
	if fi, err := os.Stdout.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		noColor = true
	}
	if noColor {
		Reset, Bold, Dim, Red, Green, Yellow, Blue, Purple, Cyan, White = "", "", "", "", "", "", "", "", "", ""
	}
}

// Logo is the ASCII art "gostrata" wordmark, one entry per line.
var Logo = []string{
	"                    _                 _",
	"  __ _   ___   ___ | |_  _ __   __ _ | |_   __ _",
	" / _` | / _ \\ / __|| __|| '__| / _` || __| / _` |",
	"| (_| || (_) |\\__ \\| |_ | |   | (_| || |_ | (_| |",
	" \\__, | \\___/ |___/ \\__||_|    \\__,_| \\__| \\__,_|",
	" |___/",
}

// LogoBlock returns the colored wordmark followed by the version.
func LogoBlock(version string) string {
	var b strings.Builder
	for i, l := range Logo {
		b.WriteString(Bold + Cyan + l + Reset)
		if i == len(Logo)-2 {
			b.WriteString("  " + Dim + White + "v" + version + Reset)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Banner prints the stylish ASCII art header.
func Banner(version string) {
	fmt.Println()
	fmt.Print(LogoBlock(version))
	fmt.Printf("%s  Go Clean Architecture Scaffolding Tool%s\n", Dim, Reset)
	fmt.Println()
}

// Success prints a green checkmark message.
func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s%s✓%s %s\n", Bold, Green, Reset, msg)
}

// Info prints a cyan info message.
func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s%sℹ%s %s\n", Bold, Cyan, Reset, msg)
}

// Warn prints a yellow warning message.
func Warn(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s%s!%s %s\n", Bold, Yellow, Reset, msg)
}

// Step prints a step indicator.
func Step(step int, total int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s[%d/%d]%s %s...\n", Cyan, step, total, Reset, msg)
}

// Error prints an error message to stderr.
func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "  %s%s✖ Error:%s %s\n", Bold, Red, Reset, msg)
}

// Header prints a section header.
func Header(title string) {
	fmt.Printf("\n%s%s%s%s\n", Bold, White, title, Reset)
}
