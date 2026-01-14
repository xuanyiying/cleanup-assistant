package visualizer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuanyiying/cleanup-cli/internal/output"
)

// DiffType represents the type of change
type DiffType int

const (
	DiffUnchanged DiffType = iota
	DiffAdded
	DiffRemoved
	DiffMoved
	DiffRenamed
)

// DiffEntry represents a single diff entry
type DiffEntry struct {
	Type    DiffType
	Path    string
	NewPath string // For moved/renamed files
	Size    int64
	IsDir   bool
}

// DiffResult represents the result of comparing two directory states
type DiffResult struct {
	Entries      []*DiffEntry
	AddedCount   int
	RemovedCount int
	MovedCount   int
	RenamedCount int
	TotalSize    int64
}

// DiffRenderer renders directory diffs
type DiffRenderer struct {
	console *output.Console
	styler  *output.Styler
	tree    *TreeVisualizer
}

// Symbols for diff display
const (
	SymbolAdded   = "+"
	SymbolRemoved = "-"
	SymbolMoved   = "→"
	SymbolRenamed = "~"
)

// NewDiffRenderer creates a new diff renderer
func NewDiffRenderer(console *output.Console) *DiffRenderer {
	styler := output.NewStyler(true) // Enable colors by default
	
	// Create a tree visualizer for rendering directory structures
	treeOptions := &TreeOptions{
		MaxDepth:   0,
		ShowSize:   true,
		ShowHidden: false,
		UseColor:   true,
		UseUnicode: true,
		IndentSize: 3,
	}
	tree := NewTreeVisualizer(console, treeOptions)
	
	return &DiffRenderer{
		console: console,
		styler:  styler,
		tree:    tree,
	}
}

// CaptureState captures the current state of a directory
func (r *DiffRenderer) CaptureState(path string) (*TreeNode, error) {
	return r.tree.BuildTree(path)
}

// Compare compares two directory states
func (r *DiffRenderer) Compare(before, after *TreeNode) *DiffResult {
	result := &DiffResult{
		Entries: make([]*DiffEntry, 0),
	}
	
	// Create maps for efficient lookup
	beforeMap := r.buildPathMap(before)
	afterMap := r.buildPathMap(after)
	
	// Find removed files (in before but not in after)
	for path, node := range beforeMap {
		if _, exists := afterMap[path]; !exists {
			entry := &DiffEntry{
				Type:  DiffRemoved,
				Path:  path,
				Size:  node.Size,
				IsDir: node.IsDir,
			}
			result.Entries = append(result.Entries, entry)
			result.RemovedCount++
			if !node.IsDir {
				result.TotalSize += node.Size
			}
		}
	}
	
	// Find added files (in after but not in before)
	for path, node := range afterMap {
		if _, exists := beforeMap[path]; !exists {
			entry := &DiffEntry{
				Type:  DiffAdded,
				Path:  path,
				Size:  node.Size,
				IsDir: node.IsDir,
			}
			result.Entries = append(result.Entries, entry)
			result.AddedCount++
			if !node.IsDir {
				result.TotalSize += node.Size
			}
		}
	}
	
	// Detect moves and renames by comparing file content/size
	// This is a simplified implementation - in practice, you might use
	// more sophisticated algorithms like content hashing
	r.detectMovesAndRenames(result, beforeMap, afterMap)
	
	// Sort entries by path for consistent output
	sort.Slice(result.Entries, func(i, j int) bool {
		return result.Entries[i].Path < result.Entries[j].Path
	})
	
	return result
}

// buildPathMap creates a map of relative paths to TreeNodes
func (r *DiffRenderer) buildPathMap(root *TreeNode) map[string]*TreeNode {
	pathMap := make(map[string]*TreeNode)
	r.buildPathMapRecursive(root, "", pathMap)
	return pathMap
}

// buildPathMapRecursive recursively builds the path map
func (r *DiffRenderer) buildPathMapRecursive(node *TreeNode, basePath string, pathMap map[string]*TreeNode) {
	// Calculate relative path
	var relativePath string
	if basePath == "" {
		relativePath = node.Name
	} else {
		relativePath = filepath.Join(basePath, node.Name)
	}
	
	pathMap[relativePath] = node
	
	// Process children
	for _, child := range node.Children {
		r.buildPathMapRecursive(child, relativePath, pathMap)
	}
}

// detectMovesAndRenames detects moved and renamed files
func (r *DiffRenderer) detectMovesAndRenames(result *DiffResult, beforeMap, afterMap map[string]*TreeNode) {
	// Create maps by size and name for potential matches
	removedBySize := make(map[int64][]*DiffEntry)
	addedBySize := make(map[int64][]*DiffEntry)
	
	// Group removed and added entries by size
	for _, entry := range result.Entries {
		if entry.Type == DiffRemoved && !entry.IsDir {
			removedBySize[entry.Size] = append(removedBySize[entry.Size], entry)
		} else if entry.Type == DiffAdded && !entry.IsDir {
			addedBySize[entry.Size] = append(addedBySize[entry.Size], entry)
		}
	}
	
	// Find potential moves/renames
	var toRemove []*DiffEntry
	
	for size, removedEntries := range removedBySize {
		if addedEntries, exists := addedBySize[size]; exists {
			// Match files with same size
			for i, removed := range removedEntries {
				if i < len(addedEntries) {
					added := addedEntries[i]
					
					// Determine if it's a move or rename
					removedDir := filepath.Dir(removed.Path)
					addedDir := filepath.Dir(added.Path)
					removedName := filepath.Base(removed.Path)
					addedName := filepath.Base(added.Path)
					
					var moveEntry *DiffEntry
					if removedDir != addedDir && removedName == addedName {
						// Same name, different directory = move
						moveEntry = &DiffEntry{
							Type:    DiffMoved,
							Path:    removed.Path,
							NewPath: added.Path,
							Size:    removed.Size,
							IsDir:   removed.IsDir,
						}
						result.MovedCount++
					} else if removedDir == addedDir && removedName != addedName {
						// Same directory, different name = rename
						moveEntry = &DiffEntry{
							Type:    DiffRenamed,
							Path:    removed.Path,
							NewPath: added.Path,
							Size:    removed.Size,
							IsDir:   removed.IsDir,
						}
						result.RenamedCount++
					} else if removedDir != addedDir && removedName != addedName {
						// Different directory and name = move + rename
						moveEntry = &DiffEntry{
							Type:    DiffMoved, // Treat as move for simplicity
							Path:    removed.Path,
							NewPath: added.Path,
							Size:    removed.Size,
							IsDir:   removed.IsDir,
						}
						result.MovedCount++
					}
					
					if moveEntry != nil {
						result.Entries = append(result.Entries, moveEntry)
						toRemove = append(toRemove, removed, added)
						
						// Adjust counts
						result.AddedCount--
						result.RemovedCount--
						result.TotalSize -= removed.Size // Remove from total since it's not actually removed
					}
				}
			}
		}
	}
	
	// Remove entries that were converted to moves/renames
	var filteredEntries []*DiffEntry
	for _, entry := range result.Entries {
		shouldRemove := false
		for _, remove := range toRemove {
			if entry == remove {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			filteredEntries = append(filteredEntries, entry)
		}
	}
	result.Entries = filteredEntries
}

// Render renders a diff result
func (r *DiffRenderer) Render(result *DiffResult) string {
	if len(result.Entries) == 0 {
		return r.styler.Dim("No changes made")
	}
	
	var builder strings.Builder
	
	// Group entries by type for better organization
	var added, removed, moved, renamed []*DiffEntry
	for _, entry := range result.Entries {
		switch entry.Type {
		case DiffAdded:
			added = append(added, entry)
		case DiffRemoved:
			removed = append(removed, entry)
		case DiffMoved:
			moved = append(moved, entry)
		case DiffRenamed:
			renamed = append(renamed, entry)
		}
	}
	
	// Render added files
	if len(added) > 0 {
		builder.WriteString(r.styler.Green("Added files:") + "\n")
		for _, entry := range added {
			symbol := r.styler.Green(SymbolAdded)
			path := entry.Path
			if entry.IsDir {
				path += "/"
			}
			sizeStr := ""
			if !entry.IsDir {
				sizeStr = fmt.Sprintf(" (%s)", formatSize(entry.Size))
			}
			builder.WriteString(fmt.Sprintf("  %s %s%s\n", symbol, path, sizeStr))
		}
		builder.WriteString("\n")
	}
	
	// Render removed files
	if len(removed) > 0 {
		builder.WriteString(r.styler.Red("Removed files:") + "\n")
		for _, entry := range removed {
			symbol := r.styler.Red(SymbolRemoved)
			path := entry.Path
			if entry.IsDir {
				path += "/"
			}
			sizeStr := ""
			if !entry.IsDir {
				sizeStr = fmt.Sprintf(" (%s)", formatSize(entry.Size))
			}
			builder.WriteString(fmt.Sprintf("  %s %s%s\n", symbol, path, sizeStr))
		}
		builder.WriteString("\n")
	}
	
	// Render moved files
	if len(moved) > 0 {
		builder.WriteString(r.styler.Yellow("Moved files:") + "\n")
		for _, entry := range moved {
			symbol := r.styler.Yellow(SymbolMoved)
			fromPath := entry.Path
			toPath := entry.NewPath
			if entry.IsDir {
				fromPath += "/"
				toPath += "/"
			}
			sizeStr := ""
			if !entry.IsDir {
				sizeStr = fmt.Sprintf(" (%s)", formatSize(entry.Size))
			}
			builder.WriteString(fmt.Sprintf("  %s %s %s %s%s\n", symbol, fromPath, symbol, toPath, sizeStr))
		}
		builder.WriteString("\n")
	}
	
	// Render renamed files
	if len(renamed) > 0 {
		builder.WriteString(r.styler.Yellow("Renamed files:") + "\n")
		for _, entry := range renamed {
			symbol := r.styler.Yellow(SymbolRenamed)
			fromPath := filepath.Base(entry.Path)
			toPath := filepath.Base(entry.NewPath)
			if entry.IsDir {
				fromPath += "/"
				toPath += "/"
			}
			sizeStr := ""
			if !entry.IsDir {
				sizeStr = fmt.Sprintf(" (%s)", formatSize(entry.Size))
			}
			builder.WriteString(fmt.Sprintf("  %s %s %s %s%s\n", symbol, fromPath, SymbolMoved, toPath, sizeStr))
		}
		builder.WriteString("\n")
	}
	
	return strings.TrimSuffix(builder.String(), "\n")
}

// RenderSideBySide renders before/after trees side by side
func (r *DiffRenderer) RenderSideBySide(before, after *TreeNode, diff *DiffResult) string {
	var builder strings.Builder
	
	// Render header
	builder.WriteString(r.styler.Bold("Directory Structure Comparison") + "\n")
	builder.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	// Render before state
	builder.WriteString(r.styler.Bold("Before:") + "\n")
	beforeStr := r.tree.Render(before)
	builder.WriteString(beforeStr)
	builder.WriteString("\n")
	
	// Render after state
	builder.WriteString(r.styler.Bold("After:") + "\n")
	afterStr := r.tree.Render(after)
	builder.WriteString(afterStr)
	builder.WriteString("\n")
	
	// Render changes
	builder.WriteString(r.styler.Bold("Changes:") + "\n")
	changesStr := r.Render(diff)
	builder.WriteString(changesStr)
	builder.WriteString("\n")
	
	// Render summary
	summaryStr := r.RenderSummary(diff)
	builder.WriteString(summaryStr)
	
	return builder.String()
}

// RenderSummary renders a summary of changes
func (r *DiffRenderer) RenderSummary(result *DiffResult) string {
	if len(result.Entries) == 0 {
		return r.styler.Dim("No changes made")
	}
	
	var builder strings.Builder
	
	builder.WriteString(r.styler.Bold("Summary:") + "\n")
	
	if result.AddedCount > 0 {
		builder.WriteString(fmt.Sprintf("  %s %d files added\n", 
			r.styler.Green(SymbolAdded), result.AddedCount))
	}
	
	if result.RemovedCount > 0 {
		builder.WriteString(fmt.Sprintf("  %s %d files removed\n", 
			r.styler.Red(SymbolRemoved), result.RemovedCount))
	}
	
	if result.MovedCount > 0 {
		builder.WriteString(fmt.Sprintf("  %s %d files moved\n", 
			r.styler.Yellow(SymbolMoved), result.MovedCount))
	}
	
	if result.RenamedCount > 0 {
		builder.WriteString(fmt.Sprintf("  %s %d files renamed\n", 
			r.styler.Yellow(SymbolRenamed), result.RenamedCount))
	}
	
	// Show total size impact
	if result.TotalSize != 0 {
		if result.TotalSize > 0 {
			builder.WriteString(fmt.Sprintf("  Space impact: %s %s\n", 
				r.styler.Green("+"), formatSize(result.TotalSize)))
		} else {
			builder.WriteString(fmt.Sprintf("  Space impact: %s %s\n", 
				r.styler.Red("-"), formatSize(-result.TotalSize)))
		}
	}
	
	return strings.TrimSuffix(builder.String(), "\n")
}