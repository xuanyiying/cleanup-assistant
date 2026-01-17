package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestBar_Basic(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewBar(100, "Testing", buf)

	bar.Add(50)
	bar.Finish()

	output := buf.String()
	if !strings.Contains(output, "Testing") {
		t.Error("Expected description in output")
	}
	if !strings.Contains(output, "100.0%") {
		t.Error("Expected 100% in final output")
	}
}

func TestBar_Set(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewBar(100, "Testing", buf)

	bar.Set(75)
	bar.Finish()

	output := buf.String()
	if !strings.Contains(output, "100/100") {
		t.Error("Expected 100/100 in final output")
	}
}

func TestBar_Overflow(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewBar(100, "Testing", buf)

	bar.Add(150) // Should cap at 100
	bar.Finish()

	output := buf.String()
	if !strings.Contains(output, "100/100") {
		t.Error("Expected progress to cap at total")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3661 * time.Second, "1h1m"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, got, tt.want)
		}
	}
}

func TestMultiBar(t *testing.T) {
	buf := &bytes.Buffer{}
	mb := NewMultiBar(buf)

	bar1 := mb.AddBar(100, "Task 1")
	bar2 := mb.AddBar(50, "Task 2")

	bar1.Add(50)
	bar2.Add(25)

	mb.Finish()

	output := buf.String()
	if !strings.Contains(output, "Task 1") {
		t.Error("Expected Task 1 in output")
	}
	if !strings.Contains(output, "Task 2") {
		t.Error("Expected Task 2 in output")
	}
}

func TestBar_RateLimit(t *testing.T) {
	buf := &bytes.Buffer{}
	bar := NewBar(1000, "Testing", buf)
	bar.updateRate = 50 * time.Millisecond

	// Rapid updates should be rate limited
	for i := 0; i < 100; i++ {
		bar.Add(1)
	}

	// Should have fewer updates than additions
	lines := strings.Count(buf.String(), "\r")
	if lines >= 100 {
		t.Error("Expected rate limiting to reduce number of updates")
	}
}
