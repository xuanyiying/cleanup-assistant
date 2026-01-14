package output

import "fmt"

// Color represents ANSI color codes
type Color int

const (
	ColorDefault Color = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

// Style represents text styling options
type Style struct {
	FgColor   Color
	BgColor   Color
	Bold      bool
	Italic    bool
	Underline bool
}

// Styler handles text styling with ANSI codes
type Styler struct {
	enabled bool
}

// NewStyler creates a new styler
func NewStyler(enabled bool) *Styler {
	return &Styler{
		enabled: enabled,
	}
}

// Apply applies style to text
func (s *Styler) Apply(text string, style Style) string {
	if !s.enabled {
		return text
	}

	var codes []string

	// Foreground color
	if style.FgColor != ColorDefault {
		codes = append(codes, fmt.Sprintf("3%d", style.FgColor-1))
	}

	// Background color
	if style.BgColor != ColorDefault {
		codes = append(codes, fmt.Sprintf("4%d", style.BgColor-1))
	}

	// Text attributes
	if style.Bold {
		codes = append(codes, "1")
	}
	if style.Italic {
		codes = append(codes, "3")
	}
	if style.Underline {
		codes = append(codes, "4")
	}

	if len(codes) == 0 {
		return text
	}

	// Build ANSI escape sequence
	var codeStr string
	for i, code := range codes {
		if i > 0 {
			codeStr += ";"
		}
		codeStr += code
	}

	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", codeStr, text)
}

// Red returns red colored text
func (s *Styler) Red(text string) string {
	return s.Apply(text, Style{FgColor: ColorRed})
}

// Green returns green colored text
func (s *Styler) Green(text string) string {
	return s.Apply(text, Style{FgColor: ColorGreen})
}

// Yellow returns yellow colored text
func (s *Styler) Yellow(text string) string {
	return s.Apply(text, Style{FgColor: ColorYellow})
}

// Blue returns blue colored text
func (s *Styler) Blue(text string) string {
	return s.Apply(text, Style{FgColor: ColorBlue})
}

// Bold returns bold text
func (s *Styler) Bold(text string) string {
	return s.Apply(text, Style{Bold: true})
}

// Dim returns dimmed text
func (s *Styler) Dim(text string) string {
	if !s.enabled {
		return text
	}
	return fmt.Sprintf("\x1b[2m%s\x1b[0m", text)
}