package cleaner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetPlatform detects and returns the current operating system
func GetPlatform() string {
	return runtime.GOOS
}

// ExpandPath expands environment variables and home directory in path
// Supports:
// - ~ for home directory (Unix-like systems)
// - %VAR% for Windows environment variables
// - $VAR for Unix environment variables
func ExpandPath(path string) string {
	// Handle home directory expansion
	if strings.HasPrefix(path, "~/") || path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if can't get home dir
		}
		if path == "~" {
			return homeDir
		}
		return filepath.Join(homeDir, path[2:])
	}

	// Handle environment variables
	// os.ExpandEnv handles both $VAR (Unix) and %VAR% (Windows) formats
	expanded := os.ExpandEnv(path)
	
	return expanded
}

// GetPathSeparator returns the platform-specific path separator
func GetPathSeparator() string {
	return string(filepath.Separator)
}

// IsProtectedPath checks if a path is protected by the operating system
// On macOS, this includes System Integrity Protection (SIP) paths
// On Windows, this includes system-critical directories
func IsProtectedPath(path string) bool {
	platform := GetPlatform()
	
	// Normalize path for comparison
	normalizedPath := filepath.Clean(path)
	
	switch platform {
	case "darwin":
		return isMacOSProtectedPath(normalizedPath)
	case "windows":
		return isWindowsProtectedPath(normalizedPath)
	default:
		// For other Unix-like systems, protect common system directories
		return isUnixProtectedPath(normalizedPath)
	}
}

// isMacOSProtectedPath checks if a path is protected by macOS SIP
func isMacOSProtectedPath(path string) bool {
	// System Integrity Protection (SIP) protected paths on macOS
	sipProtectedPaths := []string{
		"/System",
		"/usr",
		"/bin",
		"/sbin",
		"/var",
	}
	
	// SIP exceptions (these subdirectories are writable)
	sipExceptions := []string{
		"/usr/local",
		"/var/tmp",
		"/var/folders",
	}
	
	// Check if path is under an exception first
	for _, exception := range sipExceptions {
		if strings.HasPrefix(path, exception) {
			return false
		}
	}
	
	// Check if path is under a protected directory
	for _, protected := range sipProtectedPaths {
		if strings.HasPrefix(path, protected) {
			return true
		}
	}
	
	return false
}

// isWindowsProtectedPath checks if a path is protected on Windows
func isWindowsProtectedPath(path string) bool {
	// Normalize to lowercase for case-insensitive comparison
	lowerPath := strings.ToLower(path)
	
	// Windows system-critical directories
	protectedPaths := []string{
		"c:\\windows\\system32",
		"c:\\windows\\syswow64",
		"c:\\windows\\winsxs",
		"c:\\program files\\windows",
		"c:\\program files (x86)\\windows",
	}
	
	for _, protected := range protectedPaths {
		if strings.HasPrefix(lowerPath, protected) {
			return true
		}
	}
	
	return false
}

// isUnixProtectedPath checks if a path is protected on Unix-like systems
func isUnixProtectedPath(path string) bool {
	// Common protected paths on Unix-like systems
	protectedPaths := []string{
		"/boot",
		"/dev",
		"/proc",
		"/sys",
		"/etc",
		"/bin",
		"/sbin",
		"/usr/bin",
		"/usr/sbin",
		"/lib",
		"/lib64",
	}
	
	for _, protected := range protectedPaths {
		if strings.HasPrefix(path, protected) {
			return true
		}
	}
	
	return false
}
