package shell

import (
	"testing"
	"time"
)

func TestSpinnerCreation(t *testing.T) {
	spinner := NewSpinner("Testing...")
	if spinner == nil {
		t.Fatal("spinner is nil")
	}

	if spinner.message != "Testing..." {
		t.Errorf("expected message 'Testing...', got '%s'", spinner.message)
	}

	// Stop the spinner to clean up
	spinner.Stop()
}

func TestSpinnerSucceed(t *testing.T) {
	spinner := NewSpinner("Testing...")

	// Give the spinner a moment to start
	time.Sleep(50 * time.Millisecond)

	// Should not panic
	spinner.Succeed("Success!")
}

func TestSpinnerFail(t *testing.T) {
	spinner := NewSpinner("Testing...")

	// Give the spinner a moment to start
	time.Sleep(50 * time.Millisecond)

	// Should not panic
	spinner.Fail("Failed!")
}

func TestSpinnerUpdate(t *testing.T) {
	spinner := NewSpinner("Initial message")

	spinner.Update("Updated message")

	if spinner.message != "Updated message" {
		t.Errorf("expected message 'Updated message', got '%s'", spinner.message)
	}

	spinner.Stop()
}

func TestNewInteractiveShell(t *testing.T) {
	shell := NewInteractiveShell(nil, nil, nil, nil, nil, nil)
	if shell == nil {
		t.Fatal("shell is nil")
	}

	if shell.targetDir != "." {
		t.Errorf("expected targetDir '.', got '%s'", shell.targetDir)
	}
}
