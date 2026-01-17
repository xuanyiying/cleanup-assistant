package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Bar represents a progress bar
type Bar struct {
	total       int64
	current     int64
	width       int
	description string
	startTime   time.Time
	mu          sync.Mutex
	writer      io.Writer
	lastUpdate  time.Time
	updateRate  time.Duration
}

// NewBar creates a new progress bar
func NewBar(total int64, description string, writer io.Writer) *Bar {
	return &Bar{
		total:       total,
		current:     0,
		width:       50,
		description: description,
		startTime:   time.Now(),
		writer:      writer,
		lastUpdate:  time.Time{},
		updateRate:  100 * time.Millisecond, // Update at most every 100ms
	}
}

// Add increments the progress bar
func (b *Bar) Add(n int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.current += n
	if b.current > b.total {
		b.current = b.total
	}

	// Rate limit updates
	now := time.Now()
	if now.Sub(b.lastUpdate) < b.updateRate && b.current < b.total {
		return
	}
	b.lastUpdate = now

	b.render()
}

// Set sets the current progress
func (b *Bar) Set(current int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.current = current
	if b.current > b.total {
		b.current = b.total
	}

	b.render()
}

// Finish completes the progress bar
func (b *Bar) Finish() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.current = b.total
	b.render()
	fmt.Fprintln(b.writer)
}

// render draws the progress bar
func (b *Bar) render() {
	if b.writer == nil {
		return
	}

	percent := float64(b.current) / float64(b.total) * 100
	filled := int(float64(b.width) * float64(b.current) / float64(b.total))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", b.width-filled)

	elapsed := time.Since(b.startTime)
	var eta string
	if b.current > 0 {
		rate := float64(b.current) / elapsed.Seconds()
		remaining := float64(b.total-b.current) / rate
		eta = formatDuration(time.Duration(remaining) * time.Second)
	} else {
		eta = "calculating..."
	}

	// Clear line and print progress
	fmt.Fprintf(b.writer, "\r\033[K%s [%s] %.1f%% (%d/%d) ETA: %s",
		b.description, bar, percent, b.current, b.total, eta)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// MultiBar manages multiple progress bars
type MultiBar struct {
	bars   []*Bar
	mu     sync.Mutex
	writer io.Writer
}

// NewMultiBar creates a new multi-bar progress tracker
func NewMultiBar(writer io.Writer) *MultiBar {
	return &MultiBar{
		bars:   make([]*Bar, 0),
		writer: writer,
	}
}

// AddBar adds a new progress bar
func (mb *MultiBar) AddBar(total int64, description string) *Bar {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	bar := NewBar(total, description, mb.writer)
	mb.bars = append(mb.bars, bar)
	return bar
}

// Finish completes all progress bars
func (mb *MultiBar) Finish() {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	for _, bar := range mb.bars {
		bar.Finish()
	}
}
