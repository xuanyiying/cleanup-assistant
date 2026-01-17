package dedup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileInfo represents information about a file for deduplication
type FileInfo struct {
	Path     string
	Size     int64
	Hash     string
	ModTime  time.Time
	IsBackup bool // Whether this file is in a backup/trash location
}

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	Hash  string
	Size  int64
	Files []*FileInfo
}

// Deduplicator finds and manages duplicate files
type Deduplicator struct {
	// Configuration
	MinSize int64 // Minimum file size to consider (skip tiny files)
	MaxSize int64 // Maximum file size to hash (0 = no limit)
}

// NewDeduplicator creates a new deduplicator
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		MinSize: 1024,        // 1KB minimum
		MaxSize: 100 * 1024 * 1024, // 100MB maximum by default
	}
}

// FindDuplicates scans a directory and finds duplicate files
func (d *Deduplicator) FindDuplicates(ctx context.Context, rootPath string) ([]*DuplicateGroup, error) {
	// First pass: group by size
	sizeGroups := make(map[int64][]*FileInfo)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories and files outside size range
		if info.IsDir() {
			return nil
		}
		if info.Size() < d.MinSize {
			return nil
		}
		if d.MaxSize > 0 && info.Size() > d.MaxSize {
			return nil
		}

		fileInfo := &FileInfo{
			Path:     path,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			IsBackup: isBackupLocation(path),
		}

		sizeGroups[info.Size()] = append(sizeGroups[info.Size()], fileInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Second pass: hash files with same size
	hashGroups := make(map[string]*DuplicateGroup)

	for size, files := range sizeGroups {
		// Only hash if there are multiple files of the same size
		if len(files) < 2 {
			continue
		}

		for _, file := range files {
			// Check context cancellation
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			hash, err := d.hashFile(file.Path)
			if err != nil {
				continue // Skip files we can't hash
			}

			file.Hash = hash

			if group, exists := hashGroups[hash]; exists {
				group.Files = append(group.Files, file)
			} else {
				hashGroups[hash] = &DuplicateGroup{
					Hash:  hash,
					Size:  size,
					Files: []*FileInfo{file},
				}
			}
		}
	}

	// Filter to only groups with duplicates
	var duplicates []*DuplicateGroup
	for _, group := range hashGroups {
		if len(group.Files) > 1 {
			// Sort files: prefer non-backup, then newer files
			sort.Slice(group.Files, func(i, j int) bool {
				if group.Files[i].IsBackup != group.Files[j].IsBackup {
					return !group.Files[i].IsBackup // Non-backup first
				}
				return group.Files[i].ModTime.After(group.Files[j].ModTime) // Newer first
			})
			duplicates = append(duplicates, group)
		}
	}

	// Sort groups by size (largest first)
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Size > duplicates[j].Size
	})

	return duplicates, nil
}

// hashFile computes SHA-256 hash of a file
func (d *Deduplicator) hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// isBackupLocation checks if a path is in a backup/trash location
func isBackupLocation(path string) bool {
	lowerPath := strings.ToLower(filepath.ToSlash(path))
	backupIndicators := []string{
		"backup",
		"trash",
		".trash",
		"recycle",
		"old",
		"archive",
		"tmp",
		"temp",
	}

	for _, indicator := range backupIndicators {
		if strings.Contains(lowerPath, "/"+indicator+"/") ||
			strings.HasPrefix(lowerPath, indicator+"/") ||
			strings.HasSuffix(lowerPath, "/"+indicator) {
			return true
		}
	}
	return false
}

// RemovalPlan represents a plan for removing duplicate files
type RemovalPlan struct {
	Groups      []*DuplicateGroup
	ToRemove    []*FileInfo
	ToKeep      []*FileInfo
	SpaceSaved  int64
}

// CreateRemovalPlan creates a plan for removing duplicates
// keepStrategy: "newest", "oldest", "first", "manual"
func (d *Deduplicator) CreateRemovalPlan(groups []*DuplicateGroup, keepStrategy string) *RemovalPlan {
	plan := &RemovalPlan{
		Groups:   groups,
		ToRemove: make([]*FileInfo, 0),
		ToKeep:   make([]*FileInfo, 0),
	}

	for _, group := range groups {
		if len(group.Files) < 2 {
			continue
		}

		var keepIndex int
		switch keepStrategy {
		case "newest":
			keepIndex = 0 // Already sorted with newest first
		case "oldest":
			keepIndex = len(group.Files) - 1
		case "first":
			keepIndex = 0
		default:
			keepIndex = 0 // Default to newest
		}

		for i, file := range group.Files {
			if i == keepIndex {
				plan.ToKeep = append(plan.ToKeep, file)
			} else {
				plan.ToRemove = append(plan.ToRemove, file)
				plan.SpaceSaved += file.Size
			}
		}
	}

	return plan
}

// ExecuteRemovalPlan removes duplicate files according to the plan
func (d *Deduplicator) ExecuteRemovalPlan(ctx context.Context, plan *RemovalPlan, dryRun bool) error {
	for _, file := range plan.ToRemove {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if dryRun {
			fmt.Printf("Would remove: %s\n", file.Path)
			continue
		}

		if err := os.Remove(file.Path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", file.Path, err)
		}
	}

	return nil
}

// Stats returns statistics about duplicates
type Stats struct {
	TotalGroups      int
	TotalDuplicates  int
	TotalFiles       int
	WastedSpace      int64
	LargestDuplicate int64
}

// GetStats calculates statistics from duplicate groups
func GetStats(groups []*DuplicateGroup) *Stats {
	stats := &Stats{}

	for _, group := range groups {
		stats.TotalGroups++
		stats.TotalFiles += len(group.Files)
		stats.TotalDuplicates += len(group.Files) - 1
		stats.WastedSpace += group.Size * int64(len(group.Files)-1)

		if group.Size > stats.LargestDuplicate {
			stats.LargestDuplicate = group.Size
		}
	}

	return stats
}
