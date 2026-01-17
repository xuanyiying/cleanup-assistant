package analyzer

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileMetadata represents complete metadata for a file
type FileMetadata struct {
	Path              string
	Name              string
	Extension         string
	Size              int64
	MimeType          string
	CreatedAt         time.Time
	ModifiedAt        time.Time
	ContentPreview    string
	ExifData          map[string]string
	Hash              string
	FileNameQuality   FileNameQuality // 文件名质量评估
	NeedsSmarterName  bool            // 是否需要智能重命名
	SuggestedName     string          // AI 建议的文件名
	ScenarioCategory  string          // 文档场景分类（简历、面试、会议等）
	NeedsScenarioAnalysis bool        // 是否需要场景分析
}

// FileNameQuality represents the quality assessment of a filename
type FileNameQuality string

const (
	FileNameGood       FileNameQuality = "good"        // 文件名清晰有意义
	FileNameGeneric    FileNameQuality = "generic"     // 通用名称，不够具体
	FileNameMeaningless FileNameQuality = "meaningless" // 无意义的名称
	FileNameUnknown    FileNameQuality = "unknown"     // 未评估
)

// ScanOptions controls directory scanning behavior
type ScanOptions struct {
	Recursive         bool
	IncludeHidden     bool
	Filter            *FileFilter
	ExcludeExtensions []string // 要排除的文件扩展名 (不带点，如 "txt", "log")
	ExcludePatterns   []string // 要排除的文件名模式 (支持通配符)
	ExcludeDirs       []string // 要排除的目录名
	CalculateHash     bool     // 是否计算文件哈希 (默认 true)
	Workers           int      // 并发工作线程数 (默认 4)
}

// FileFilter defines criteria for filtering files
type FileFilter struct {
	Patterns       []string
	MinSize        int64
	MaxSize        int64
	ModifiedAfter  time.Time
	ModifiedBefore time.Time
}

// Analyzer defines the interface for file analysis operations
type Analyzer interface {
	Analyze(ctx context.Context, path string) (*FileMetadata, error)
	AnalyzeDirectory(ctx context.Context, path string, opts *ScanOptions) ([]*FileMetadata, error)
	DetectType(path string) (string, error)
	AssessFileNameQuality(filename string) FileNameQuality
}

// FileAnalyzer implements the Analyzer interface
type FileAnalyzer struct{}

// NewAnalyzer creates a new file analyzer
func NewAnalyzer() *FileAnalyzer {
	return &FileAnalyzer{}
}

// Analyze extracts complete metadata for a single file
func (fa *FileAnalyzer) Analyze(ctx context.Context, path string) (*FileMetadata, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Extract basic metadata
	name := info.Name()
	ext := filepath.Ext(name)
	if ext != "" {
		ext = ext[1:] // Remove leading dot
	}

	// Detect MIME type
	mimeType, err := fa.DetectType(path)
	if err != nil {
		mimeType = "application/octet-stream"
	}

	// Calculate file hash (can be skipped for performance)
	hash := ""

	// Extract content preview for text files
	preview := ""
	if strings.HasPrefix(mimeType, "text/") {
		preview = fa.extractTextPreview(path, 500) // Increased from 200 to 500
	}

	metadata := &FileMetadata{
		Path:             path,
		Name:             name,
		Extension:        ext,
		Size:             info.Size(),
		MimeType:         mimeType,
		CreatedAt:        info.ModTime(), // Go doesn't have creation time on all platforms
		ModifiedAt:       info.ModTime(),
		ContentPreview:   preview,
		ExifData:         make(map[string]string),
		Hash:             hash,
		FileNameQuality:  fa.AssessFileNameQuality(name),
		NeedsSmarterName: false,
		SuggestedName:    "",
		ScenarioCategory: "",
		NeedsScenarioAnalysis: false,
	}

	// 判断是否需要智能重命名
	if metadata.FileNameQuality == FileNameMeaningless || metadata.FileNameQuality == FileNameGeneric {
		metadata.NeedsSmarterName = true
	}

	// 判断是否需要场景分析（文档类型）
	if strings.HasPrefix(mimeType, "text/") || 
		strings.Contains(mimeType, "pdf") ||
		strings.Contains(mimeType, "document") ||
		strings.Contains(mimeType, "word") ||
		strings.Contains(mimeType, "excel") {
		metadata.NeedsScenarioAnalysis = true
	}

	return metadata, nil
}

// AnalyzeDirectory scans a directory and returns metadata for all matching files
// Performance optimized with concurrent file analysis
func (fa *FileAnalyzer) AnalyzeDirectory(ctx context.Context, path string, opts *ScanOptions) ([]*FileMetadata, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if opts == nil {
		opts = &ScanOptions{
			Recursive:     true,
			IncludeHidden: false,
			CalculateHash: true,
			Workers:       4,
		}
	}

	// Set defaults
	if opts.Workers <= 0 {
		opts.Workers = 4
	}

	// Phase 1: Collect file paths (fast, sequential)
	var filePaths []string
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory should be excluded
			if fa.shouldExcludeDir(info.Name(), opts) {
				return filepath.SkipDir
			}
			
			// If not recursive, skip subdirectories
			if !opts.Recursive && filePath != path {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files if requested
		if !opts.IncludeHidden && strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check if file should be excluded
		if fa.shouldExcludeFile(info.Name(), opts) {
			return nil
		}

		// Apply filters
		if opts.Filter != nil {
			if !fa.matchesFilter(filePath, info, opts.Filter) {
				return nil
			}
		}

		filePaths = append(filePaths, filePath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Phase 2: Analyze files concurrently (slow, parallel)
	return fa.analyzeFilesConcurrently(ctx, filePaths, opts)
}

// analyzeFilesConcurrently analyzes multiple files using a worker pool
func (fa *FileAnalyzer) analyzeFilesConcurrently(ctx context.Context, filePaths []string, opts *ScanOptions) ([]*FileMetadata, error) {
	if len(filePaths) == 0 {
		return []*FileMetadata{}, nil
	}

	// Create channels
	fileChan := make(chan string, len(filePaths))
	resultChan := make(chan *FileMetadata, len(filePaths))
	errorChan := make(chan error, opts.Workers)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				select {
				case <-ctx.Done():
					errorChan <- ctx.Err()
					return
				default:
				}

				metadata, err := fa.analyzeWithOptions(ctx, filePath, opts)
				if err != nil {
					// Skip files that can't be analyzed
					continue
				}
				resultChan <- metadata
			}
		}()
	}

	// Send files to workers
	go func() {
		for _, filePath := range filePaths {
			fileChan <- filePath
		}
		close(fileChan)
	}()

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results
	var results []*FileMetadata
	for metadata := range resultChan {
		results = append(results, metadata)
	}

	// Check for errors
	select {
	case err := <-errorChan:
		if err != nil {
			return results, err
		}
	default:
	}

	return results, nil
}

// analyzeWithOptions analyzes a file with specific options (hash calculation, etc.)
func (fa *FileAnalyzer) analyzeWithOptions(ctx context.Context, path string, opts *ScanOptions) (*FileMetadata, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Extract basic metadata
	name := info.Name()
	ext := filepath.Ext(name)
	if ext != "" {
		ext = ext[1:] // Remove leading dot
	}

	// Detect MIME type
	mimeType, err := fa.DetectType(path)
	if err != nil {
		mimeType = "application/octet-stream"
	}

	// Calculate file hash only if requested
	hash := ""
	if opts.CalculateHash {
		hash, _ = fa.calculateHash(path)
	}

	// Extract content preview for text files
	preview := ""
	if strings.HasPrefix(mimeType, "text/") {
		preview = fa.extractTextPreview(path, 500)
	}

	metadata := &FileMetadata{
		Path:             path,
		Name:             name,
		Extension:        ext,
		Size:             info.Size(),
		MimeType:         mimeType,
		CreatedAt:        info.ModTime(),
		ModifiedAt:       info.ModTime(),
		ContentPreview:   preview,
		ExifData:         make(map[string]string),
		Hash:             hash,
		FileNameQuality:  fa.AssessFileNameQuality(name),
		NeedsSmarterName: false,
		SuggestedName:    "",
		ScenarioCategory: "",
		NeedsScenarioAnalysis: false,
	}

	// 判断是否需要智能重命名
	if metadata.FileNameQuality == FileNameMeaningless || metadata.FileNameQuality == FileNameGeneric {
		metadata.NeedsSmarterName = true
	}

	// 判断是否需要场景分析（文档类型）
	if strings.HasPrefix(mimeType, "text/") || 
		strings.Contains(mimeType, "pdf") ||
		strings.Contains(mimeType, "document") ||
		strings.Contains(mimeType, "word") ||
		strings.Contains(mimeType, "excel") {
		metadata.NeedsScenarioAnalysis = true
	}

	return metadata, nil
}

// DetectType detects the MIME type of a file using magic bytes and extension
func (fa *FileAnalyzer) DetectType(path string) (string, error) {
	// First try to detect by magic bytes
	mimeType, err := fa.detectByMagicBytes(path)
	if err == nil && mimeType != "" {
		return mimeType, nil
	}

	// Fall back to extension-based detection
	ext := filepath.Ext(path)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType, nil
		}
	}

	return "application/octet-stream", nil
}

// detectByMagicBytes detects MIME type by reading file magic bytes
func (fa *FileAnalyzer) detectByMagicBytes(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read first 512 bytes for magic byte detection
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return "", err
	}

	header = header[:n]

	// Common magic bytes
	if len(header) >= 4 {
		// JPEG
		if header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
			return "image/jpeg", nil
		}
		// PNG
		if header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 {
			return "image/png", nil
		}
		// GIF
		if header[0] == 0x47 && header[1] == 0x49 && header[2] == 0x46 {
			return "image/gif", nil
		}
		// PDF
		if header[0] == 0x25 && header[1] == 0x50 && header[2] == 0x44 && header[3] == 0x46 {
			return "application/pdf", nil
		}
		// ZIP (including docx, xlsx, etc.)
		if header[0] == 0x50 && header[1] == 0x4B && header[2] == 0x03 && header[3] == 0x04 {
			return "application/zip", nil
		}
	}

	if len(header) >= 2 {
		// UTF-16 BOM
		if header[0] == 0xFF && header[1] == 0xFE {
			return "text/plain; charset=utf-16", nil
		}
		if header[0] == 0xFE && header[1] == 0xFF {
			return "text/plain; charset=utf-16", nil
		}
	}

	// Check if it's text
	if fa.isTextFile(header) {
		return "text/plain", nil
	}

	return "", nil
}

// isTextFile checks if content appears to be text
func (fa *FileAnalyzer) isTextFile(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// Check for null bytes (common in binary files)
	for _, b := range data {
		if b == 0 {
			return false
		}
	}

	// If mostly printable ASCII, consider it text
	printable := 0
	for _, b := range data {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printable++
		}
	}

	return float64(printable)/float64(len(data)) > 0.75
}

// calculateHash computes MD5 hash of file content
func (fa *FileAnalyzer) calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// extractTextPreview reads first N characters of a text file
func (fa *FileAnalyzer) extractTextPreview(path string, maxChars int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	// For text files, try to read more content for better analysis
	readSize := maxChars
	if strings.HasSuffix(strings.ToLower(path), ".txt") ||
		strings.HasSuffix(strings.ToLower(path), ".md") ||
		strings.HasSuffix(strings.ToLower(path), ".doc") {
		readSize = maxChars * 3 // Read more for documents
	}

	data := make([]byte, readSize)
	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		return ""
	}

	// Clean up preview: remove null bytes and control characters
	preview := string(data[:n])
	preview = strings.Map(func(r rune) rune {
		if r == 0 || (r < 32 && r != '\n' && r != '\r' && r != '\t') {
			return -1
		}
		return r
	}, preview)

	preview = strings.TrimSpace(preview)
	
	// If preview is too long, truncate to maxChars but try to end at word boundary
	if len(preview) > maxChars {
		preview = preview[:maxChars]
		// Find last space to avoid cutting words
		if lastSpace := strings.LastIndex(preview, " "); lastSpace > maxChars/2 {
			preview = preview[:lastSpace]
		}
		preview = preview + "..."
	}

	return preview
}

// matchesFilter checks if a file matches the given filter criteria
func (fa *FileAnalyzer) matchesFilter(filePath string, info os.FileInfo, filter *FileFilter) bool {
	if filter == nil {
		return true
	}

	// Check size constraints
	if filter.MinSize > 0 && info.Size() < filter.MinSize {
		return false
	}
	if filter.MaxSize > 0 && info.Size() > filter.MaxSize {
		return false
	}

	// Check date constraints
	modTime := info.ModTime()
	if !filter.ModifiedAfter.IsZero() && modTime.Before(filter.ModifiedAfter) {
		return false
	}
	if !filter.ModifiedBefore.IsZero() && modTime.After(filter.ModifiedBefore) {
		return false
	}

	// Check patterns
	if len(filter.Patterns) > 0 {
		matched := false
		for _, pattern := range filter.Patterns {
			if match, _ := filepath.Match(pattern, filepath.Base(filePath)); match {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// AssessFileNameQuality evaluates the quality of a filename
// Returns whether the filename is meaningful, generic, or meaningless
func (fa *FileAnalyzer) AssessFileNameQuality(filename string) FileNameQuality {
	// Remove extension for analysis
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	nameWithoutExt = strings.TrimSpace(nameWithoutExt)

	if nameWithoutExt == "" {
		return FileNameMeaningless
	}

	// Convert to lowercase for comparison
	nameLower := strings.ToLower(nameWithoutExt)

	// Meaningless patterns: random characters, timestamps, generic names
	meaninglessPatterns := []string{
		"untitled", "新建", "新建文档", "无标题", "new file", "document",
		"image", "photo", "picture", "download",
		"screenshot", "屏幕截图", "截图", "img", "pic",
		"temp", "tmp", "test", "copy", "副本",
	}

	for _, pattern := range meaninglessPatterns {
		if nameLower == pattern || strings.HasPrefix(nameLower, pattern+" ") || strings.HasPrefix(nameLower, pattern) {
			return FileNameMeaningless
		}
	}

	// Check for timestamp-only names (e.g., "20240101_123456", "IMG_1234")
	if isTimestampLikeName(nameLower) {
		return FileNameMeaningless
	}

	// Check for very short names (1-2 characters)
	if len(nameWithoutExt) <= 2 {
		return FileNameMeaningless
	}

	// Generic patterns: common prefixes without meaningful content
	genericPatterns := []string{
		"doc", "data", "report", "notes",
	}

	for _, pattern := range genericPatterns {
		if nameLower == pattern {
			return FileNameGeneric
		}
	}

	// Special case: "file" is meaningless
	if nameLower == "file" {
		return FileNameMeaningless
	}

	// Check if name is mostly numbers
	digitCount := 0
	for _, r := range nameWithoutExt {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}
	if float64(digitCount)/float64(len(nameWithoutExt)) > 0.7 {
		return FileNameMeaningless
	}

	// If name has meaningful length and content, consider it good
	if len(nameWithoutExt) >= 3 {
		return FileNameGood
	}

	return FileNameUnknown
}

// shouldExcludeFile checks if a file should be excluded based on options
func (fa *FileAnalyzer) shouldExcludeFile(filename string, opts *ScanOptions) bool {
	if opts == nil {
		return false
	}

	// Check extension exclusions
	if len(opts.ExcludeExtensions) > 0 {
		ext := filepath.Ext(filename)
		if ext != "" {
			ext = strings.ToLower(ext[1:]) // Remove leading dot and lowercase
			for _, excludeExt := range opts.ExcludeExtensions {
				if strings.ToLower(excludeExt) == ext {
					return true
				}
			}
		}
	}

	// Check pattern exclusions
	if len(opts.ExcludePatterns) > 0 {
		for _, pattern := range opts.ExcludePatterns {
			matched, err := filepath.Match(pattern, filename)
			if err == nil && matched {
				return true
			}
			// Also try case-insensitive match
			matched, err = filepath.Match(strings.ToLower(pattern), strings.ToLower(filename))
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// shouldExcludeDir checks if a directory should be excluded based on options
func (fa *FileAnalyzer) shouldExcludeDir(dirname string, opts *ScanOptions) bool {
	if opts == nil || len(opts.ExcludeDirs) == 0 {
		return false
	}

	for _, excludeDir := range opts.ExcludeDirs {
		if dirname == excludeDir {
			return true
		}
		// Also try case-insensitive match for common patterns
		if strings.EqualFold(dirname, excludeDir) {
			return true
		}
	}

	return false
}

// isTimestampLikeName checks if a filename looks like a timestamp or auto-generated ID
func isTimestampLikeName(name string) bool {
	// Common timestamp patterns
	timestampPatterns := []string{
		"img_", "dsc_", "pic_", "photo_", "screenshot_",
		"wechat", "mmexport", "wx_camera_",
	}

	for _, pattern := range timestampPatterns {
		if strings.HasPrefix(name, pattern) {
			// Check if followed by mostly numbers
			suffix := strings.TrimPrefix(name, pattern)
			digitCount := 0
			for _, r := range suffix {
				if r >= '0' && r <= '9' {
					digitCount++
				}
			}
			if float64(digitCount)/float64(len(suffix)) > 0.6 {
				return true
			}
		}
	}

	// Check for pure numeric or date-like patterns
	// e.g., "20240101", "123456", "2024-01-01"
	cleaned := strings.ReplaceAll(name, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "_", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	digitCount := 0
	for _, r := range cleaned {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}

	return float64(digitCount)/float64(len(cleaned)) > 0.8
}
