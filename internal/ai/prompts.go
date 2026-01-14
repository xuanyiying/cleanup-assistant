package ai

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xuanyiying/cleanup-cli/internal/analyzer"
)

// GenerateNameSuggestionPrompt creates a prompt for file name suggestion
func GenerateNameSuggestionPrompt(file *analyzer.FileMetadata) string {
	if file == nil {
		return ""
	}

	// Check if it's a document type
	isDocument := strings.HasPrefix(file.MimeType, "text/") ||
		strings.Contains(file.MimeType, "pdf") ||
		strings.Contains(file.MimeType, "document") ||
		strings.Contains(file.MimeType, "word") ||
		strings.Contains(file.MimeType, "excel") ||
		strings.Contains(file.MimeType, "powerpoint")

	if file.ContentPreview != "" && len(file.ContentPreview) > 20 {
		if isDocument {
			return fmt.Sprintf(
				"You are a file naming expert. Analyze the document content and create a descriptive filename.\n\n"+
					"Document content:\n%s\n\n"+
					"Requirements:\n"+
					"1. Extract the main topic or purpose from the content\n"+
					"2. Create a clear, specific filename (e.g., 'quarterly-sales-report-2024', 'team-meeting-notes-jan')\n"+
					"3. Use only lowercase letters, numbers, and hyphens\n"+
					"4. Keep it between 15-50 characters\n"+
					"5. Do NOT include file extension\n"+
					"6. Do NOT use quotes or special characters\n\n"+
					"Output ONLY the filename:",
				file.ContentPreview,
			)
		} else {
			return fmt.Sprintf(
				"Create a descriptive filename based on this content:\n\n%s\n\n"+
					"Rules:\n"+
					"- Use lowercase with hyphens\n"+
					"- Be specific and concise\n"+
					"- Maximum 40 characters\n"+
					"- No file extension\n\n"+
					"Filename:",
				file.ContentPreview,
			)
		}
	}

	return fmt.Sprintf(
		"Suggest a better filename for: %s (type: %s)\n\n"+
			"Output a concise, descriptive name using lowercase and hyphens. "+
			"No extension. Maximum 30 characters.\n\n"+
			"Filename:",
		file.Name, file.MimeType,
	)
}

// CleanSuggestedName cleans up the AI suggested filename
func CleanSuggestedName(suggestedName string) string {
	suggestedName = strings.TrimSpace(suggestedName)

	// Take first line only
	lines := strings.Split(suggestedName, "\n")
	suggestedName = strings.TrimSpace(lines[0])

	// Remove common prefixes that AI might add
	prefixes := []string{"Filename:", "filename:", "Suggested:", "suggested:"}
	for _, p := range prefixes {
		suggestedName = strings.TrimPrefix(suggestedName, p)
	}
	suggestedName = strings.TrimSpace(suggestedName)

	// Remove quotes and backticks
	suggestedName = strings.Trim(suggestedName, "\"'`")

	// Remove any file extension if AI added one
	if ext := filepath.Ext(suggestedName); ext != "" {
		suggestedName = strings.TrimSuffix(suggestedName, ext)
	}

	// Clean up: replace spaces with hyphens, convert to lowercase
	suggestedName = strings.ToLower(suggestedName)
	suggestedName = strings.ReplaceAll(suggestedName, " ", "-")
	suggestedName = strings.ReplaceAll(suggestedName, "_", "-")

	// Remove multiple consecutive hyphens
	for strings.Contains(suggestedName, "--") {
		suggestedName = strings.ReplaceAll(suggestedName, "--", "-")
	}

	// Trim hyphens from start and end
	suggestedName = strings.Trim(suggestedName, "-")

	return suggestedName
}

// GenerateCategorySuggestionPrompt creates a prompt for category suggestion
func GenerateCategorySuggestionPrompt(file *analyzer.FileMetadata) string {
	if file == nil {
		return ""
	}

	// If no content preview, return empty string to indicate skipping
	if file.ContentPreview == "" || len(file.ContentPreview) < 20 {
		return ""
	}

	return fmt.Sprintf(
		"Analyze the document content and categorize it by its purpose/scenario.\n\n"+
			"Document content:\n%s\n\n"+
			"Based on the content, identify the document's primary purpose/scenario.\n"+
			"Return ONE category from these options:\n"+
			"- resume (简历、CV、个人简历)\n"+
			"- interview (面试题、面试准备、面试笔记)\n"+
			"- meeting (会议记录、会议纪要、讨论记录)\n"+
			"- report (报告、分析报告、工作报告)\n"+
			"- proposal (提案、建议书、项目提案)\n"+
			"- contract (合同、协议、条款)\n"+
			"- invoice (发票、账单、收据)\n"+
			"- guide (指南、教程、说明书)\n"+
			"- notes (笔记、备忘录、草稿)\n"+
			"- other (其他)\n\n"+
			"Output ONLY the category name (lowercase):",
		file.ContentPreview,
	)
}

// CleanSuggestedCategory cleans up the AI suggested category
func CleanSuggestedCategory(category string) string {
	category = strings.TrimSpace(category)
	category = strings.ToLower(category)

	// Take first line only
	lines := strings.Split(category, "\n")
	category = strings.TrimSpace(lines[0])

	// Remove common prefixes
	prefixes := []string{"category:", "Category:"}
	for _, p := range prefixes {
		category = strings.TrimPrefix(category, p)
	}
	category = strings.TrimSpace(category)

	// Remove quotes
	category = strings.Trim(category, "\"'`")

	// Validate category
	validCategories := map[string]bool{
		"resume":    true,
		"interview": true,
		"meeting":   true,
		"report":    true,
		"proposal":  true,
		"contract":  true,
		"invoice":   true,
		"guide":     true,
		"notes":     true,
		"other":     true,
	}

	if !validCategories[category] {
		return "other"
	}

	return category
}
