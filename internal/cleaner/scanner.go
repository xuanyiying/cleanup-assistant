package cleaner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// JunkCategory represents a category of junk files
type JunkCategory string

const (
	CategoryCache     JunkCategory = "cache"
	CategoryLogs      JunkCategory = "logs"
	CategoryTemp      JunkCategory = "temp"
	CategoryTrash     JunkCategory = "trash"
	CategoryBrowser   JunkCategory = "browser"
	CategoryDeveloper JunkCategory = "developer"
	CategorySystem    JunkCategory = "system"
)

// JunkLocation represents a junk file location
type JunkLocation struct {
	Path        string
	Category    JunkCategory
	Description string
	Platform    string // "darwin", "windows", "all"
	Pattern     string // Optional glob pattern
}

// JunkFile represents a detected junk file
type JunkFile struct {
	Path        string
	Size        int64
	Category    JunkCategory
	ModTime     time.Time
	IsImportant bool // If true, requires confirmation
}

// ScanResult represents the result of a junk scan
type ScanResult struct {
	Files       []*JunkFile
	TotalSize   int64
	ByCategory  map[JunkCategory][]*JunkFile
	Skipped     []string // Paths skipped due to permissions
	Errors      []error
}

// JunkScanner scans for junk files
type JunkScanner struct {
	locations  []*JunkLocation
	classifier *FileClassifier
	platform   string
}

// NewJunkScanner creates a new junk scanner
func NewJunkScanner() *JunkScanner {
	scanner := &JunkScanner{
		locations:  []*JunkLocation{},
		classifier: NewFileClassifier(),
		platform:   runtime.GOOS,
	}

	// Add default locations for current platform
	for _, location := range scanner.GetDefaultLocations() {
		scanner.AddLocation(location)
	}

	return scanner
}

// AddLocation adds a custom junk location
func (s *JunkScanner) AddLocation(loc *JunkLocation) {
	s.locations = append(s.locations, loc)
}

// ClearLocations clears all junk locations
func (s *JunkScanner) ClearLocations() {
	s.locations = []*JunkLocation{}
}

// Scan scans for junk files
func (s *JunkScanner) Scan(ctx context.Context) (*ScanResult, error) {
	result := &ScanResult{
		Files:      []*JunkFile{},
		TotalSize:  0,
		ByCategory: make(map[JunkCategory][]*JunkFile),
		Skipped:    []string{},
		Errors:     []error{},
	}

	for _, location := range s.locations {
		// Skip locations not for current platform
		if location.Platform != "all" && location.Platform != s.platform {
			continue
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Expand environment variables in path
		expandedPath := s.expandPath(location.Path)
		
		// Scan this location
		locationFiles, skipped, errors := s.scanLocation(expandedPath, location)
		
		// Add files to result
		for _, file := range locationFiles {
			// Check if file is important and should be excluded
			if s.classifier.IsImportant(file.Path) {
				file.IsImportant = true
				// Skip important files by default (Requirements 6.5)
				continue
			}

			result.Files = append(result.Files, file)
			result.TotalSize += file.Size
			
			// Add to category map
			if result.ByCategory[file.Category] == nil {
				result.ByCategory[file.Category] = []*JunkFile{}
			}
			result.ByCategory[file.Category] = append(result.ByCategory[file.Category], file)
		}

		// Add skipped paths and errors
		result.Skipped = append(result.Skipped, skipped...)
		result.Errors = append(result.Errors, errors...)
	}

	return result, nil
}

// ScanCategory scans only a specific category
func (s *JunkScanner) ScanCategory(ctx context.Context, category JunkCategory) (*ScanResult, error) {
	result := &ScanResult{
		Files:      []*JunkFile{},
		TotalSize:  0,
		ByCategory: make(map[JunkCategory][]*JunkFile),
		Skipped:    []string{},
		Errors:     []error{},
	}

	for _, location := range s.locations {
		// Skip locations not for current platform or category
		if location.Platform != "all" && location.Platform != s.platform {
			continue
		}
		if location.Category != category {
			continue
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Expand environment variables in path
		expandedPath := s.expandPath(location.Path)
		
		// Scan this location
		locationFiles, skipped, errors := s.scanLocation(expandedPath, location)
		
		// Add files to result
		for _, file := range locationFiles {
			// Check if file is important and should be excluded
			if s.classifier.IsImportant(file.Path) {
				file.IsImportant = true
				// Skip important files by default (Requirements 6.5)
				continue
			}

			result.Files = append(result.Files, file)
			result.TotalSize += file.Size
			
			// Add to category map
			if result.ByCategory[file.Category] == nil {
				result.ByCategory[file.Category] = []*JunkFile{}
			}
			result.ByCategory[file.Category] = append(result.ByCategory[file.Category], file)
		}

		// Add skipped paths and errors
		result.Skipped = append(result.Skipped, skipped...)
		result.Errors = append(result.Errors, errors...)
	}

	return result, nil
}

// GetDefaultLocations returns default junk locations for current platform
func (s *JunkScanner) GetDefaultLocations() []*JunkLocation {
	switch s.platform {
	case "darwin":
		return s.getMacOSJunkLocations()
	case "windows":
		return s.getWindowsJunkLocations()
	default:
		// Return common locations for other platforms
		return []*JunkLocation{
			{Path: "/tmp", Category: CategoryTemp, Description: "Temporary files", Platform: "all"},
			{Path: "~/.cache", Category: CategoryCache, Description: "User cache", Platform: "all"},
		}
	}
}

// getMacOSJunkLocations returns macOS-specific junk locations
func (s *JunkScanner) getMacOSJunkLocations() []*JunkLocation {
	return []*JunkLocation{
		{Path: "~/Library/Caches", Category: CategoryCache, Description: "User caches", Platform: "darwin"},
		{Path: "~/.cache", Category: CategoryCache, Description: "Hidden cache", Platform: "darwin"},
		{Path: "~/Library/Logs", Category: CategoryLogs, Description: "User logs", Platform: "darwin"},
		{Path: "/var/log", Category: CategoryLogs, Description: "System logs", Platform: "darwin"},
		{Path: "/tmp", Category: CategoryTemp, Description: "Temporary files", Platform: "darwin"},
		{Path: "/var/tmp", Category: CategoryTemp, Description: "System temp", Platform: "darwin"},
		{Path: "~/.Trash", Category: CategoryTrash, Description: "User trash", Platform: "darwin"},
		{Path: "~/Library/Developer/Xcode/DerivedData", Category: CategoryDeveloper, Description: "Xcode derived data", Platform: "darwin"},
		{Path: "~/Library/Application Support/MobileSync/Backup", Category: CategoryDeveloper, Description: "iOS backups", Platform: "darwin"},
		{Path: "~/Library/Caches/com.apple.Safari", Category: CategoryBrowser, Description: "Safari cache", Platform: "darwin"},
		{Path: "~/Library/Caches/Google/Chrome", Category: CategoryBrowser, Description: "Chrome cache", Platform: "darwin"},
		{Path: "~/Library/Caches/Firefox", Category: CategoryBrowser, Description: "Firefox cache", Platform: "darwin"},
	}
}

// getWindowsJunkLocations returns Windows-specific junk locations
func (s *JunkScanner) getWindowsJunkLocations() []*JunkLocation {
	return []*JunkLocation{
		{Path: "%TEMP%", Category: CategoryTemp, Description: "User temp files", Platform: "windows"},
		{Path: "%TMP%", Category: CategoryTemp, Description: "System temp files", Platform: "windows"},
		{Path: "C:\\Windows\\Temp", Category: CategoryTemp, Description: "Windows temp", Platform: "windows"},
		{Path: "C:\\Windows\\Prefetch", Category: CategorySystem, Description: "Prefetch files", Platform: "windows"},
		{Path: "%LOCALAPPDATA%\\Microsoft\\Windows\\Explorer", Category: CategoryCache, Description: "Thumbnail cache", Platform: "windows"},
		{Path: "C:\\Windows\\SoftwareDistribution\\Download", Category: CategorySystem, Description: "Windows Update cache", Platform: "windows"},
		{Path: "%LOCALAPPDATA%\\Google\\Chrome\\User Data\\Default\\Cache", Category: CategoryBrowser, Description: "Chrome cache", Platform: "windows"},
		{Path: "%LOCALAPPDATA%\\Mozilla\\Firefox\\Profiles\\*\\cache2", Category: CategoryBrowser, Description: "Firefox cache", Platform: "windows"},
		{Path: "%LOCALAPPDATA%\\Microsoft\\Edge\\User Data\\Default\\Cache", Category: CategoryBrowser, Description: "Edge cache", Platform: "windows"},
	}
}

// scanLocation scans a specific location for junk files
func (s *JunkScanner) scanLocation(path string, location *JunkLocation) ([]*JunkFile, []string, []error) {
	var files []*JunkFile
	var skipped []string
	var errors []error

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Path doesn't exist, not an error for junk scanning
		return files, skipped, errors
	}

	// Walk the directory
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Permission denied or other access error
			if os.IsPermission(err) {
				skipped = append(skipped, filePath)
				return nil // Continue walking
			}
			errors = append(errors, fmt.Errorf("error accessing %s: %w", filePath, err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Create junk file entry
		junkFile := &JunkFile{
			Path:        filePath,
			Size:        info.Size(),
			Category:    location.Category,
			ModTime:     info.ModTime(),
			IsImportant: false,
		}

		files = append(files, junkFile)
		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Errorf("error walking %s: %w", path, err))
	}

	return files, skipped, errors
}

// expandPath expands environment variables and home directory in path
func (s *JunkScanner) expandPath(path string) string {
	// Handle home directory expansion
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if can't get home dir
		}
		return filepath.Join(homeDir, path[2:])
	}

	// Handle environment variables
	return os.ExpandEnv(path)
}