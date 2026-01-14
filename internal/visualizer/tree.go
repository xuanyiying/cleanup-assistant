package visualizer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuanyiying/cleanup-cli/internal/output"
)

// TreeNode represents a node in the directory tree
type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Children []*TreeNode
	Depth    int
}

// TreeOptions configures tree rendering
type TreeOptions struct {
	MaxDepth   int  // Maximum depth to display (0 = unlimited)
	ShowSize   bool // Show file sizes
	ShowHidden bool // Show hidden files
	UseColor   bool // Use ANSI colors
	UseUnicode bool // Use Unicode box characters
	IndentSize int  // Spaces per indent level
}

// TreeVisualizer renders directory trees
type TreeVisualizer struct {
	console *output.Console
	styler  *output.Styler
	options *TreeOptions
}

// Branch characters for tree rendering
const (
	BranchVertical   = "│"
	BranchHorizontal = "──"
	BranchCorner     = "└"
	BranchTee        = "├"
	BranchEmpty      = "   "

	// ASCII fallback
	BranchVerticalASCII   = "|"
	BranchHorizontalASCII = "--"
	BranchCornerASCII     = "`"
	BranchTeeASCII        = "+"
)

// NewTreeVisualizer creates a new tree visualizer
func NewTreeVisualizer(console *output.Console, options *TreeOptions) *TreeVisualizer {
	if options == nil {
		options = &TreeOptions{
			MaxDepth:   0,
			ShowSize:   false,
			ShowHidden: false,
			UseColor:   true,
			UseUnicode: true,
			IndentSize: 3,
		}
	}

	// Create styler based on color preference
	styler := output.NewStyler(options.UseColor)

	return &TreeVisualizer{
		console: console,
		styler:  styler,
		options: options,
	}
}

// BuildTree builds a tree structure from a directory path
func (v *TreeVisualizer) BuildTree(path string) (*TreeNode, error) {
	return v.buildTreeRecursive(path, 0)
}

// buildTreeRecursive recursively builds the tree structure
func (v *TreeVisualizer) buildTreeRecursive(path string, depth int) (*TreeNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &TreeNode{
		Name:  filepath.Base(path),
		Path:  path,
		IsDir: info.IsDir(),
		Size:  info.Size(),
		Depth: depth,
	}

	// If it's a file or we've reached max depth, return the node
	if !info.IsDir() || (v.options.MaxDepth > 0 && depth >= v.options.MaxDepth) {
		return node, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		// If we can't read the directory, return the node without children
		return node, nil
	}

	// Filter and sort entries
	var filteredEntries []os.DirEntry
	for _, entry := range entries {
		// Skip hidden files if not showing them
		if !v.options.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	// Sort entries: directories first, then alphabetically
	sort.Slice(filteredEntries, func(i, j int) bool {
		iInfo, _ := filteredEntries[i].Info()
		jInfo, _ := filteredEntries[j].Info()
		
		if iInfo.IsDir() != jInfo.IsDir() {
			return iInfo.IsDir() // Directories first
		}
		return filteredEntries[i].Name() < filteredEntries[j].Name()
	})

	// Build children
	for _, entry := range filteredEntries {
		childPath := filepath.Join(path, entry.Name())
		child, err := v.buildTreeRecursive(childPath, depth+1)
		if err != nil {
			// Skip entries we can't access
			continue
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

// Render renders a tree node to string
func (v *TreeVisualizer) Render(node *TreeNode) string {
	var builder strings.Builder
	v.renderNode(node, "", true, &builder)
	return builder.String()
}

// RenderToWriter renders a tree node to an io.Writer
func (v *TreeVisualizer) RenderToWriter(node *TreeNode, w io.Writer) error {
	content := v.Render(node)
	_, err := w.Write([]byte(content))
	return err
}

// renderNode recursively renders a node and its children
func (v *TreeVisualizer) renderNode(node *TreeNode, prefix string, isLast bool, builder *strings.Builder) {
	// Choose branch characters based on Unicode setting
	var vertical, horizontal, corner, tee string
	if v.options.UseUnicode {
		vertical = BranchVertical
		horizontal = BranchHorizontal
		corner = BranchCorner
		tee = BranchTee
	} else {
		vertical = BranchVerticalASCII
		horizontal = BranchHorizontalASCII
		corner = BranchCornerASCII
		tee = BranchTeeASCII
	}

	// Build the current line
	var line strings.Builder
	
	// Add prefix from parent levels
	line.WriteString(prefix)
	
	// Add branch character for current level
	if node.Depth > 0 {
		if isLast {
			line.WriteString(corner + horizontal + " ")
		} else {
			line.WriteString(tee + horizontal + " ")
		}
	}

	// Add the node name with appropriate styling
	name := node.Name
	if node.IsDir {
		if v.options.UseColor {
			name = v.styler.Blue(name)
		}
		name += "/"
	}
	line.WriteString(name)

	// Add size if requested
	if v.options.ShowSize && !node.IsDir {
		sizeStr := formatSize(node.Size)
		if v.options.UseColor {
			sizeStr = v.styler.Dim(fmt.Sprintf(" (%s)", sizeStr))
		} else {
			sizeStr = fmt.Sprintf(" (%s)", sizeStr)
		}
		line.WriteString(sizeStr)
	}

	// Add empty indicator for empty directories
	if node.IsDir && len(node.Children) == 0 {
		emptyStr := "(empty)"
		if v.options.UseColor {
			emptyStr = v.styler.Dim(emptyStr)
		}
		line.WriteString(" " + emptyStr)
	}

	builder.WriteString(line.String())
	builder.WriteString("\n")

	// Render children
	for i, child := range node.Children {
		isChildLast := i == len(node.Children)-1
		
		// Build prefix for child
		var childPrefix string
		if node.Depth >= 0 {
			if isLast {
				childPrefix = prefix + strings.Repeat(" ", v.options.IndentSize)
			} else {
				childPrefix = prefix + vertical + strings.Repeat(" ", v.options.IndentSize-1)
			}
		}
		
		v.renderNode(child, childPrefix, isChildLast, builder)
	}
}

// formatSize formats file size in human-readable format
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}