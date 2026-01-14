package cleaner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xuanyiying/cleanup-cli/internal/output"
)

// PromptAction represents user's choice
type PromptAction int

const (
	ActionYes PromptAction = iota
	ActionNo
	ActionAllYes
	ActionSkipAll
	ActionView
)

// String returns the string representation of PromptAction
func (a PromptAction) String() string {
	switch a {
	case ActionYes:
		return "yes"
	case ActionNo:
		return "no"
	case ActionAllYes:
		return "all_yes"
	case ActionSkipAll:
		return "skip_all"
	case ActionView:
		return "view"
	default:
		return "unknown"
	}
}

// PromptOptions configures prompt behavior
type PromptOptions struct {
	Timeout       time.Duration // Timeout for response (0 = no timeout)
	DefaultAction PromptAction  // Default action on timeout
}

// FilePrompt represents a prompt for a single file
type FilePrompt struct {
	Path    string
	Size    int64
	Type    string
	ModTime time.Time
	Preview string // First 500 chars for text files
	Reason  string // Why confirmation is needed
}

// InteractivePrompt handles user confirmation prompts
type InteractivePrompt struct {
	console *output.Console
	reader  io.Reader
	scanner *bufio.Scanner
	options *PromptOptions
	allYes  bool // User selected "All yes"
	skipAll bool // User selected "Skip all"
}

// NewInteractivePrompt creates a new interactive prompt
func NewInteractivePrompt(console *output.Console, reader io.Reader) *InteractivePrompt {
	if reader == nil {
		reader = os.Stdin
	}

	return &InteractivePrompt{
		console: console,
		reader:  reader,
		scanner: bufio.NewScanner(reader),
		options: &PromptOptions{
			Timeout:       0, // No timeout by default
			DefaultAction: ActionNo,
		},
	}
}

// SetOptions sets the prompt options
func (p *InteractivePrompt) SetOptions(options *PromptOptions) {
	if options != nil {
		p.options = options
	}
}

// Prompt prompts the user for a single file
func (p *InteractivePrompt) Prompt(file *FilePrompt) (PromptAction, error) {
	// Check batch state first
	if p.allYes {
		return ActionYes, nil
	}
	if p.skipAll {
		return ActionNo, nil
	}

	// Display file information
	p.displayFileInfo(file)

	// Get user input
	for {
		p.console.Info("What would you like to do? [Y]es, [N]o, [A]ll yes, [S]kip all, [V]iew content:")

		input, err := p.readInput()
		if err != nil {
			return p.options.DefaultAction, err
		}

		action := p.parseInput(input)
		switch action {
		case ActionYes:
			return ActionYes, nil
		case ActionNo:
			return ActionNo, nil
		case ActionAllYes:
			p.allYes = true
			return ActionYes, nil
		case ActionSkipAll:
			p.skipAll = true
			return ActionNo, nil
		case ActionView:
			if err := p.ShowPreview(file.Path, 500); err != nil {
				p.console.Error("Failed to show preview: %v", err)
			}
			// Continue the loop to ask again
		default:
			p.console.Warning("Invalid input. Please enter Y, N, A, S, or V.")
		}
	}
}

// PromptBatch prompts for multiple files with batch options
func (p *InteractivePrompt) PromptBatch(files []*FilePrompt) (map[string]PromptAction, error) {
	results := make(map[string]PromptAction)

	for _, file := range files {
		action, err := p.Prompt(file)
		if err != nil {
			return results, err
		}
		results[file.Path] = action
	}

	return results, nil
}

// ShowPreview shows file content preview
func (p *InteractivePrompt) ShowPreview(path string, maxChars int) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// Read up to maxChars
	buffer := make([]byte, maxChars)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("cannot read file: %w", err)
	}

	content := string(buffer[:n])

	// Check if content is likely text
	if !isTextContent(content) {
		p.console.Info("File appears to be binary content (showing first %d bytes as hex):", n)
		p.displayHexPreview(buffer[:n])
	} else {
		p.console.Info("File preview (first %d characters):", len(content))
		p.console.Box("File Content", []string{content})
	}

	// Show if file was truncated
	if n == maxChars {
		stat, err := file.Stat()
		if err == nil && stat.Size() > int64(maxChars) {
			p.console.Info("... (file truncated, showing first %d of %d bytes)", maxChars, stat.Size())
		}
	}

	return nil
}

// Reset resets the batch state (allYes, skipAll)
func (p *InteractivePrompt) Reset() {
	p.allYes = false
	p.skipAll = false
}

// displayFileInfo displays information about the file being prompted
func (p *InteractivePrompt) displayFileInfo(file *FilePrompt) {
	p.console.Warning("Uncertain file detected:")

	info := []string{
		fmt.Sprintf("Path: %s", file.Path),
		fmt.Sprintf("Size: %s", formatFileSize(file.Size)),
		fmt.Sprintf("Type: %s", file.Type),
		fmt.Sprintf("Modified: %s", file.ModTime.Format("2006-01-02 15:04:05")),
	}

	if file.Reason != "" {
		info = append(info, fmt.Sprintf("Reason: %s", file.Reason))
	}

	p.console.Box("File Details", info)
}

// readInput reads user input with optional timeout
func (p *InteractivePrompt) readInput() (string, error) {
	if p.options.Timeout > 0 {
		// TODO: Implement timeout functionality
		// For now, just read without timeout
	}

	if p.scanner.Scan() {
		return strings.TrimSpace(p.scanner.Text()), nil
	}

	if err := p.scanner.Err(); err != nil {
		return "", err
	}

	// EOF or empty input
	return "", io.EOF
}

// parseInput parses user input into PromptAction
func (p *InteractivePrompt) parseInput(input string) PromptAction {
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "y", "yes":
		return ActionYes
	case "n", "no":
		return ActionNo
	case "a", "all", "all yes":
		return ActionAllYes
	case "s", "skip", "skip all":
		return ActionSkipAll
	case "v", "view":
		return ActionView
	default:
		return PromptAction(-1) // Invalid
	}
}

// isTextContent checks if content appears to be text
func isTextContent(content string) bool {
	// Simple heuristic: if more than 95% of characters are printable, consider it text
	printable := 0
	for _, r := range content {
		if r >= 32 && r <= 126 || r == '\n' || r == '\r' || r == '\t' {
			printable++
		}
	}

	if len(content) == 0 {
		return true
	}

	return float64(printable)/float64(len(content)) > 0.95
}

// displayHexPreview displays binary content as hex
func (p *InteractivePrompt) displayHexPreview(data []byte) {
	const bytesPerLine = 16
	lines := []string{}

	for i := 0; i < len(data); i += bytesPerLine {
		end := i + bytesPerLine
		if end > len(data) {
			end = len(data)
		}

		// Format hex bytes
		hex := make([]string, bytesPerLine)
		ascii := make([]byte, bytesPerLine)

		for j := 0; j < bytesPerLine; j++ {
			if i+j < len(data) {
				b := data[i+j]
				hex[j] = fmt.Sprintf("%02x", b)
				if b >= 32 && b <= 126 {
					ascii[j] = b
				} else {
					ascii[j] = '.'
				}
			} else {
				hex[j] = "  "
				ascii[j] = ' '
			}
		}

		line := fmt.Sprintf("%08x  %s  %s", i,
			strings.Join(hex[:8], " ")+" "+strings.Join(hex[8:], " "),
			string(ascii))
		lines = append(lines, line)

		// Limit preview to reasonable size
		if len(lines) >= 10 {
			lines = append(lines, "...")
			break
		}
	}

	p.console.Box("Hex Preview", lines)
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}
