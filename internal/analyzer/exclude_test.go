package analyzer

import (
	"testing"
)

func TestShouldExcludeFile(t *testing.T) {
	fa := NewAnalyzer()

	tests := []struct {
		name     string
		filename string
		opts     *ScanOptions
		want     bool
	}{
		{
			name:     "exclude by extension",
			filename: "test.log",
			opts: &ScanOptions{
				ExcludeExtensions: []string{"log", "tmp"},
			},
			want: true,
		},
		{
			name:     "not excluded by extension",
			filename: "test.txt",
			opts: &ScanOptions{
				ExcludeExtensions: []string{"log", "tmp"},
			},
			want: false,
		},
		{
			name:     "exclude by pattern",
			filename: "test.bak",
			opts: &ScanOptions{
				ExcludePatterns: []string{"*.bak", "*.swp"},
			},
			want: true,
		},
		{
			name:     "exclude by pattern with wildcard",
			filename: "temp_file.txt",
			opts: &ScanOptions{
				ExcludePatterns: []string{"temp*"},
			},
			want: true,
		},
		{
			name:     "not excluded by pattern",
			filename: "document.pdf",
			opts: &ScanOptions{
				ExcludePatterns: []string{"*.bak"},
			},
			want: false,
		},
		{
			name:     "case insensitive extension",
			filename: "test.LOG",
			opts: &ScanOptions{
				ExcludeExtensions: []string{"log"},
			},
			want: true,
		},
		{
			name:     "no exclusions",
			filename: "test.txt",
			opts:     &ScanOptions{},
			want:     false,
		},
		{
			name:     "nil options",
			filename: "test.txt",
			opts:     nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fa.shouldExcludeFile(tt.filename, tt.opts)
			if got != tt.want {
				t.Errorf("shouldExcludeFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExcludeDir(t *testing.T) {
	fa := NewAnalyzer()

	tests := []struct {
		name    string
		dirname string
		opts    *ScanOptions
		want    bool
	}{
		{
			name:    "exclude .git",
			dirname: ".git",
			opts: &ScanOptions{
				ExcludeDirs: []string{".git", "node_modules"},
			},
			want: true,
		},
		{
			name:    "exclude node_modules",
			dirname: "node_modules",
			opts: &ScanOptions{
				ExcludeDirs: []string{".git", "node_modules"},
			},
			want: true,
		},
		{
			name:    "not excluded",
			dirname: "src",
			opts: &ScanOptions{
				ExcludeDirs: []string{".git", "node_modules"},
			},
			want: false,
		},
		{
			name:    "case insensitive",
			dirname: "Node_Modules",
			opts: &ScanOptions{
				ExcludeDirs: []string{"node_modules"},
			},
			want: true,
		},
		{
			name:    "no exclusions",
			dirname: "src",
			opts:    &ScanOptions{},
			want:    false,
		},
		{
			name:    "nil options",
			dirname: "src",
			opts:    nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fa.shouldExcludeDir(tt.dirname, tt.opts)
			if got != tt.want {
				t.Errorf("shouldExcludeDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
