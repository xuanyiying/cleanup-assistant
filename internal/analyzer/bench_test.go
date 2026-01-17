package analyzer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkAnalyze(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	analyzer := NewAnalyzer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(ctx, testFile)
	}
}

func BenchmarkAnalyzeDirectory(b *testing.B) {
	tmpDir := b.TempDir()
	for i := 0; i < 100; i++ {
		os.WriteFile(filepath.Join(tmpDir, "file"+string(rune(i))+".txt"), []byte("test"), 0644)
	}

	analyzer := NewAnalyzer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeDirectory(ctx, tmpDir, &ScanOptions{
			Recursive:     false,
			CalculateHash: false,
			Workers:       4,
		})
	}
}

func BenchmarkConcurrentScanning(b *testing.B) {
	tmpDir := b.TempDir()
	for i := 0; i < 100; i++ {
		os.WriteFile(filepath.Join(tmpDir, "file"+string(rune(i))+".txt"), []byte("test"), 0644)
	}

	analyzer := NewAnalyzer()
	ctx := context.Background()

	for _, workers := range []int{1, 2, 4, 8} {
		b.Run("workers_"+string(rune(workers+'0')), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				analyzer.AnalyzeDirectory(ctx, tmpDir, &ScanOptions{
					Workers:       workers,
					CalculateHash: false,
				})
			}
		})
	}
}
