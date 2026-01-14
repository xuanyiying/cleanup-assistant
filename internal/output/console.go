package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ColorSupport represents terminal color capability
type ColorSupport int

const (
	ColorNone  ColorSupport = iota // No color support
	ColorBasic                     // Basic 16 colors
	ColorFull                      // Full 256/true color
)

// Console handles all terminal output with color support
type Console struct {
	colorEnabled bool
	colorSupport ColorSupport
	writer       io.Writer
	styler       *Styler
}

// NewConsole creates a new console output handler
func NewConsole(writer io.Writer) *Console {
	console := &Console{
		writer: writer,
	}
	
	console.colorSupport = console.DetectColorSupport()
	console.colorEnabled = console.colorSupport != ColorNone
	console.styler = NewStyler(console.colorEnabled)
	
	return console
}

// DetectColorSupport detects terminal color capability
func (c *Console) DetectColorSupport() ColorSupport {
	// Check if output is a terminal
	if f, ok := c.writer.(*os.File); ok {
		if !isTerminal(f) {
			return ColorNone
		}
	}
	
	// Check environment variables
	term := os.Getenv("TERM")
	colorTerm := os.Getenv("COLORTERM")
	
	// No color support
	if term == "dumb" || os.Getenv("NO_COLOR") != "" {
		return ColorNone
	}
	
	// Full color support
	if colorTerm == "truecolor" || colorTerm == "24bit" ||
		strings.Contains(term, "256color") || strings.Contains(term, "truecolor") {
		return ColorFull
	}
	
	// Basic color support
	if strings.Contains(term, "color") || term == "xterm" || term == "screen" {
		return ColorBasic
	}
	
	return ColorNone
}

// SetColorEnabled enables or disables color output
func (c *Console) SetColorEnabled(enabled bool) {
	c.colorEnabled = enabled && c.colorSupport != ColorNone
	c.styler = NewStyler(c.colorEnabled)
}

// Success prints a success message with green ✓
func (c *Console) Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	symbol := "✓"
	if !c.colorEnabled {
		symbol = "[OK]"
	}
	
	output := fmt.Sprintf("%s %s", c.styler.Green(symbol), message)
	fmt.Fprintln(c.writer, output)
}

// Error prints an error message with red ✗
func (c *Console) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	symbol := "✗"
	if !c.colorEnabled {
		symbol = "[ERROR]"
	}
	
	output := fmt.Sprintf("%s %s", c.styler.Red(symbol), message)
	fmt.Fprintln(c.writer, output)
}

// Warning prints a warning message with yellow ⚠
func (c *Console) Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	symbol := "⚠"
	if !c.colorEnabled {
		symbol = "[WARN]"
	}
	
	output := fmt.Sprintf("%s %s", c.styler.Yellow(symbol), message)
	fmt.Fprintln(c.writer, output)
}

// Info prints an info message
func (c *Console) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(c.writer, message)
}

// Box prints content in a bordered box
func (c *Console) Box(title string, content []string) {
	if len(content) == 0 {
		return
	}
	
	// Calculate box width
	maxWidth := len(title)
	for _, line := range content {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	
	// Add padding
	boxWidth := maxWidth + 4
	
	// Box drawing characters
	topLeft := "┌"
	topRight := "┐"
	bottomLeft := "└"
	bottomRight := "┘"
	horizontal := "─"
	vertical := "│"
	
	// ASCII fallback
	if !c.colorEnabled {
		topLeft = "+"
		topRight = "+"
		bottomLeft = "+"
		bottomRight = "+"
		horizontal = "-"
		vertical = "|"
	}
	
	// Top border
	fmt.Fprintf(c.writer, "%s%s%s\n", topLeft, strings.Repeat(horizontal, boxWidth-2), topRight)
	
	// Title
	if title != "" {
		padding := boxWidth - len(title) - 2
		leftPad := padding / 2
		rightPad := padding - leftPad
		fmt.Fprintf(c.writer, "%s%s%s%s%s\n", 
			vertical, 
			strings.Repeat(" ", leftPad), 
			c.styler.Bold(title), 
			strings.Repeat(" ", rightPad), 
			vertical)
		
		// Separator
		fmt.Fprintf(c.writer, "%s%s%s\n", vertical, strings.Repeat(horizontal, boxWidth-2), vertical)
	}
	
	// Content
	for _, line := range content {
		padding := boxWidth - len(line) - 2
		fmt.Fprintf(c.writer, "%s %s%s %s\n", 
			vertical, 
			line, 
			strings.Repeat(" ", padding), 
			vertical)
	}
	
	// Bottom border
	fmt.Fprintf(c.writer, "%s%s%s\n", bottomLeft, strings.Repeat(horizontal, boxWidth-2), bottomRight)
}

// Table prints data in a formatted table
func (c *Console) Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	
	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}
	
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	
	// Table drawing characters
	topLeft := "┌"
	topRight := "┐"
	bottomLeft := "└"
	bottomRight := "┘"
	cross := "┼"
	teeDown := "┬"
	teeUp := "┴"
	teeRight := "├"
	teeLeft := "┤"
	horizontal := "─"
	vertical := "│"
	
	// ASCII fallback
	if !c.colorEnabled {
		topLeft = "+"
		topRight = "+"
		bottomLeft = "+"
		bottomRight = "+"
		cross = "+"
		teeDown = "+"
		teeUp = "+"
		teeRight = "+"
		teeLeft = "+"
		horizontal = "-"
		vertical = "|"
	}
	
	// Top border
	fmt.Fprint(c.writer, topLeft)
	for i, width := range colWidths {
		fmt.Fprint(c.writer, strings.Repeat(horizontal, width+2))
		if i < len(colWidths)-1 {
			fmt.Fprint(c.writer, teeDown)
		}
	}
	fmt.Fprintln(c.writer, topRight)
	
	// Headers
	fmt.Fprint(c.writer, vertical)
	for i, header := range headers {
		fmt.Fprintf(c.writer, " %s%s ", 
			c.styler.Bold(header), 
			strings.Repeat(" ", colWidths[i]-len(header)))
		fmt.Fprint(c.writer, vertical)
	}
	fmt.Fprintln(c.writer)
	
	// Header separator
	fmt.Fprint(c.writer, teeRight)
	for i, width := range colWidths {
		fmt.Fprint(c.writer, strings.Repeat(horizontal, width+2))
		if i < len(colWidths)-1 {
			fmt.Fprint(c.writer, cross)
		}
	}
	fmt.Fprintln(c.writer, teeLeft)
	
	// Rows
	for _, row := range rows {
		fmt.Fprint(c.writer, vertical)
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Fprintf(c.writer, " %s%s ", 
					cell, 
					strings.Repeat(" ", colWidths[i]-len(cell)))
			}
			fmt.Fprint(c.writer, vertical)
		}
		fmt.Fprintln(c.writer)
	}
	
	// Bottom border
	fmt.Fprint(c.writer, bottomLeft)
	for i, width := range colWidths {
		fmt.Fprint(c.writer, strings.Repeat(horizontal, width+2))
		if i < len(colWidths)-1 {
			fmt.Fprint(c.writer, teeUp)
		}
	}
	fmt.Fprintln(c.writer, bottomRight)
}

// isTerminal checks if the file is a terminal
func isTerminal(f *os.File) bool {
	// Simple check - in a real implementation, you might use
	// a library like golang.org/x/term for more robust detection
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}