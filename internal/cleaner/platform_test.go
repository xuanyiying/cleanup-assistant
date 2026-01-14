package cleaner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestPlatformPathSeparator tests Property 21: Platform Path Separator
// Feature: enhanced-output-cleanup, Property 21: Platform Path Separator
func TestPlatformPathSeparator(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Get the path separator
		separator := GetPathSeparator()

		// Verify it matches the current platform
		currentPlatform := runtime.GOOS

		switch currentPlatform {
		case "windows":
			assert.Equal(t, "\\", separator,
				"Windows should use backslash as path separator")
		case "darwin", "linux":
			assert.Equal(t, "/", separator,
				"Unix-like systems should use forward slash as path separator")
		default:
			// For other platforms, just verify it's one of the valid separators
			assert.True(t, separator == "/" || separator == "\\",
				"Path separator should be either / or \\")
		}

		// Verify it matches filepath.Separator
		assert.Equal(t, string(filepath.Separator), separator,
			"GetPathSeparator should return the same value as filepath.Separator")

		// Generate random path components and verify separator usage
		numComponents := rapid.IntRange(2, 5).Draw(rt, "numComponents")
		components := make([]string, numComponents)
		for i := 0; i < numComponents; i++ {
			components[i] = rapid.StringMatching(`[a-zA-Z0-9_-]+`).Draw(rt, "component")
		}

		// Join path using filepath.Join
		joinedPath := filepath.Join(components...)

		// Verify the joined path contains the correct separator
		if numComponents > 1 {
			assert.Contains(t, joinedPath, separator,
				"Joined path should contain the platform-specific separator")
		}
	})
}

// Unit tests for platform detection and path handling

func TestGetPlatform(t *testing.T) {
	platform := GetPlatform()

	assert.NotEmpty(t, platform)
	assert.Equal(t, runtime.GOOS, platform,
		"GetPlatform should return the same value as runtime.GOOS")
}

func TestExpandPath_HomeDirectory(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde with path",
			input:    "~/test/path",
			expected: filepath.Join(homeDir, "test", "path"),
		},
		{
			name:     "tilde only",
			input:    "~",
			expected: homeDir,
		},
		{
			name:     "no tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandPath_EnvironmentVariables(t *testing.T) {
	// Set test environment variable
	testVar := "TEST_EXPAND_VAR"
	testValue := "test_value"
	os.Setenv(testVar, testValue)
	defer os.Unsetenv(testVar)

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "Unix style variable",
			input:    "$TEST_EXPAND_VAR/path",
			contains: testValue,
		},
	}

	// Add Windows test only if on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			input    string
			contains string
		}{
			name:     "Windows style variable",
			input:    "%TEST_EXPAND_VAR%/path",
			contains: testValue,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			assert.Contains(t, result, tt.contains,
				"Expanded path should contain the environment variable value")
		})
	}
}

func TestExpandPath_NoExpansion(t *testing.T) {
	// Test paths that should not be expanded
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "absolute path",
			input: "/usr/local/bin",
		},
		{
			name:  "relative path",
			input: "relative/path",
		},
		{
			name:  "current directory",
			input: "./current",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			// For paths without variables or ~, result should be the same or only env-expanded
			// We just verify it doesn't panic and returns something
			assert.NotEmpty(t, result)
		})
	}
}

func TestIsProtectedPath_MacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-macOS platform")
	}

	tests := []struct {
		name      string
		path      string
		protected bool
	}{
		{
			name:      "System directory",
			path:      "/System/Library",
			protected: true,
		},
		{
			name:      "usr directory",
			path:      "/usr/bin",
			protected: true,
		},
		{
			name:      "usr/local exception",
			path:      "/usr/local/bin",
			protected: false,
		},
		{
			name:      "var/tmp exception",
			path:      "/var/tmp",
			protected: false,
		},
		{
			name:      "home directory",
			path:      os.Getenv("HOME"),
			protected: false,
		},
		{
			name:      "bin directory",
			path:      "/bin",
			protected: true,
		},
		{
			name:      "sbin directory",
			path:      "/sbin",
			protected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProtectedPath(tt.path)
			assert.Equal(t, tt.protected, result,
				"Path %s protection status should be %v", tt.path, tt.protected)
		})
	}
}

func TestIsProtectedPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	tests := []struct {
		name      string
		path      string
		protected bool
	}{
		{
			name:      "System32 directory",
			path:      "C:\\Windows\\System32",
			protected: true,
		},
		{
			name:      "SysWOW64 directory",
			path:      "C:\\Windows\\SysWOW64",
			protected: true,
		},
		{
			name:      "WinSxS directory",
			path:      "C:\\Windows\\WinSxS",
			protected: true,
		},
		{
			name:      "User directory",
			path:      "C:\\Users\\TestUser",
			protected: false,
		},
		{
			name:      "Program Files",
			path:      "C:\\Program Files\\MyApp",
			protected: false,
		},
		{
			name:      "Windows Program Files",
			path:      "C:\\Program Files\\Windows Defender",
			protected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProtectedPath(tt.path)
			assert.Equal(t, tt.protected, result,
				"Path %s protection status should be %v", tt.path, tt.protected)
		})
	}
}

func TestIsProtectedPath_Unix(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on non-Unix platform")
	}

	tests := []struct {
		name      string
		path      string
		protected bool
	}{
		{
			name:      "boot directory",
			path:      "/boot",
			protected: true,
		},
		{
			name:      "dev directory",
			path:      "/dev",
			protected: true,
		},
		{
			name:      "proc directory",
			path:      "/proc",
			protected: true,
		},
		{
			name:      "sys directory",
			path:      "/sys",
			protected: true,
		},
		{
			name:      "home directory",
			path:      "/home/user",
			protected: false,
		},
		{
			name:      "tmp directory",
			path:      "/tmp",
			protected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProtectedPath(tt.path)
			assert.Equal(t, tt.protected, result,
				"Path %s protection status should be %v", tt.path, tt.protected)
		})
	}
}

func TestIsProtectedPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "empty path",
			path: "",
		},
		{
			name: "relative path",
			path: "relative/path",
		},
		{
			name: "current directory",
			path: ".",
		},
		{
			name: "parent directory",
			path: "..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := IsProtectedPath(tt.path)
			// Just verify it returns a boolean
			assert.IsType(t, false, result)
		})
	}
}

func TestExpandPath_CombinedExpansion(t *testing.T) {
	// Test combining home directory and environment variables
	testVar := "TEST_COMBINED_VAR"
	testValue := "combined_value"
	os.Setenv(testVar, testValue)
	defer os.Unsetenv(testVar)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	// Test home directory expansion first
	input := "~/test"
	result := ExpandPath(input)
	expected := filepath.Join(homeDir, "test")
	assert.Equal(t, expected, result)

	// Test environment variable expansion
	input = "$TEST_COMBINED_VAR/path"
	result = ExpandPath(input)
	assert.Contains(t, result, testValue)
}

func TestGetPathSeparator_Consistency(t *testing.T) {
	// Test that GetPathSeparator is consistent across multiple calls
	sep1 := GetPathSeparator()
	sep2 := GetPathSeparator()

	assert.Equal(t, sep1, sep2, "GetPathSeparator should return consistent results")
	assert.Equal(t, string(filepath.Separator), sep1,
		"GetPathSeparator should match filepath.Separator")
}
