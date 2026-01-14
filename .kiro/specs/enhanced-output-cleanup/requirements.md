# Requirements Document

## Introduction

本功能扩展 Cleanup CLI，增加三个核心能力：

1. 任务完成后美化显示目录结构变化（修改前/后对比）
2. 系统垃圾文件扫描与一键清理（支持 macOS 和 Windows）
3. 重要文件高亮提示与不确定文件交互式确认

## Glossary

- **Directory_Visualizer**: 目录结构可视化模块，负责生成树形目录展示
- **Diff_Renderer**: 差异渲染器，负责对比并高亮显示变化
- **System_Cleaner**: 系统垃圾清理模块，负责扫描和清理系统垃圾文件
- **Junk_Scanner**: 垃圾文件扫描器，识别系统垃圾文件
- **File_Classifier**: 文件分类器，识别重要文件和不确定文件
- **Interactive_Prompt**: 交互式提示模块，处理用户确认

## Requirements

### Requirement 1: 目录结构可视化

**User Story:** As a user, I want to see a visual tree representation of directory structure, so that I can understand the file organization at a glance.

#### Acceptance Criteria

1. WHEN displaying a directory structure, THE Directory_Visualizer SHALL render files and folders in a tree format with proper indentation and branch characters (├── └── │)
2. WHEN the directory contains nested folders, THE Directory_Visualizer SHALL display the hierarchy with correct depth indicators
3. WHEN rendering the tree, THE Directory_Visualizer SHALL use color coding to distinguish folders (blue) from files (default)
4. WHEN a directory is empty, THE Directory_Visualizer SHALL display "(empty)" indicator
5. THE Directory_Visualizer SHALL support configurable maximum depth for display

### Requirement 2: 修改前后对比显示

**User Story:** As a user, I want to see a side-by-side or sequential comparison of directory structure before and after cleanup operations, so that I can verify what changes were made.

#### Acceptance Criteria

1. WHEN an organize operation completes, THE Diff_Renderer SHALL capture and display the directory structure before the operation
2. WHEN an organize operation completes, THE Diff_Renderer SHALL display the directory structure after the operation
3. WHEN displaying changes, THE Diff_Renderer SHALL highlight added files/folders in green with "+" prefix
4. WHEN displaying changes, THE Diff_Renderer SHALL highlight removed files/folders in red with "-" prefix
5. WHEN displaying changes, THE Diff_Renderer SHALL highlight moved files in yellow with "→" indicator showing source and destination
6. WHEN no changes were made, THE Diff_Renderer SHALL display "No changes made" message
7. THE Diff_Renderer SHALL display a summary showing counts of files added, removed, moved, and renamed

### Requirement 3: 美化控制台输出

**User Story:** As a user, I want the CLI output to be visually appealing and easy to read, so that I can quickly understand the results.

#### Acceptance Criteria

1. THE Cleanup_CLI SHALL use ANSI color codes for terminal output with fallback for non-color terminals
2. WHEN displaying progress, THE Cleanup_CLI SHALL use animated spinners and progress bars
3. WHEN displaying success messages, THE Cleanup_CLI SHALL use green color with ✓ checkmark
4. WHEN displaying error messages, THE Cleanup_CLI SHALL use red color with ✗ mark
5. WHEN displaying warnings, THE Cleanup_CLI SHALL use yellow color with ⚠ symbol
6. THE Cleanup_CLI SHALL use box drawing characters to create visual sections and borders
7. WHEN the terminal does not support colors, THE Cleanup_CLI SHALL gracefully degrade to plain text output

### Requirement 4: 系统垃圾文件扫描

**User Story:** As a user, I want to scan for system junk files on my computer, so that I can identify files that can be safely deleted to free up space.

#### Acceptance Criteria

1. WHEN scanning on macOS, THE Junk_Scanner SHALL identify common junk files including:
   - Cache files (~/.cache, ~/Library/Caches)
   - Log files (~/Library/Logs, /var/log)
   - Temporary files (/tmp, /var/tmp, ~/Library/Application Support/\*/Cache)
   - Trash files (~/.Trash)
   - Xcode derived data (~/Library/Developer/Xcode/DerivedData)
   - iOS device backups (~/Library/Application Support/MobileSync/Backup)
   - Browser caches (Safari, Chrome, Firefox)
2. WHEN scanning on Windows, THE Junk_Scanner SHALL identify common junk files including:
   - Temp files (%TEMP%, %TMP%)
   - Windows temp (C:\Windows\Temp)
   - Prefetch files (C:\Windows\Prefetch)
   - Thumbnail cache (C:\Users\*\AppData\Local\Microsoft\Windows\Explorer)
   - Windows Update cache (C:\Windows\SoftwareDistribution\Download)
   - Browser caches (Chrome, Firefox, Edge)
   - Recycle Bin
3. WHEN scanning completes, THE Junk_Scanner SHALL display total size of junk files found
4. WHEN scanning, THE Junk_Scanner SHALL categorize junk files by type (cache, logs, temp, etc.)
5. THE Junk_Scanner SHALL support custom junk file patterns via configuration
6. WHEN a junk file location requires elevated permissions, THE Junk_Scanner SHALL skip it and note in the report

### Requirement 5: 一键清理功能

**User Story:** As a user, I want to clean up identified junk files with a single command, so that I can quickly free up disk space.

#### Acceptance Criteria

1. WHEN the user runs cleanup command, THE System_Cleaner SHALL display a summary of files to be deleted before proceeding
2. WHEN cleaning, THE System_Cleaner SHALL require user confirmation before deleting files
3. WHEN cleaning, THE System_Cleaner SHALL move files to trash by default instead of permanent deletion
4. WHEN the --force flag is provided, THE System_Cleaner SHALL permanently delete files without moving to trash
5. WHEN cleaning completes, THE System_Cleaner SHALL display total space freed
6. WHEN a file cannot be deleted, THE System_Cleaner SHALL log the error and continue with remaining files
7. THE System_Cleaner SHALL support selective cleaning by category (e.g., only caches, only logs)
8. WHEN cleaning system files on macOS, THE System_Cleaner SHALL handle SIP-protected locations gracefully

### Requirement 6: 重要文件识别与提示

**User Story:** As a user, I want the tool to identify and highlight important files, so that I don't accidentally delete or move critical data.

#### Acceptance Criteria

1. WHEN scanning files, THE File_Classifier SHALL identify important files based on:
   - File patterns (_.key, _.pem, _.env, _.credentials, id_rsa, id_ed25519)
   - Directory patterns (Documents, Desktop, important, backup)
   - File size (files larger than configurable threshold)
   - Recent modification (files modified within configurable days)
2. WHEN displaying important files, THE File_Classifier SHALL use red/bold highlighting with ⚠ warning symbol
3. WHEN an operation would affect an important file, THE Cleanup_CLI SHALL display a prominent warning
4. THE File_Classifier SHALL support custom important file patterns via configuration
5. WHEN important files are found in junk scan, THE Junk_Scanner SHALL exclude them from cleanup by default

### Requirement 7: 不确定文件交互式确认

**User Story:** As a user, I want to be prompted for confirmation when the tool encounters files it's unsure about, so that I can make informed decisions.

#### Acceptance Criteria

1. WHEN the File_Classifier cannot determine if a file is safe to process, THE Interactive_Prompt SHALL ask the user for confirmation
2. WHEN prompting, THE Interactive_Prompt SHALL display file details including name, size, type, and last modified date
3. WHEN prompting, THE Interactive_Prompt SHALL offer options: [Y]es, [N]o, [A]ll yes, [S]kip all, [V]iew content
4. WHEN the user selects [V]iew content, THE Interactive_Prompt SHALL display file preview (first 500 characters for text files)
5. WHEN the user selects [A]ll yes, THE Interactive_Prompt SHALL apply the action to all remaining uncertain files
6. WHEN the user selects [S]kip all, THE Interactive_Prompt SHALL skip all remaining uncertain files
7. WHEN running in non-interactive mode (--yes flag), THE Cleanup_CLI SHALL skip uncertain files by default
8. THE Interactive_Prompt SHALL timeout after configurable seconds and default to skip

### Requirement 8: 跨平台支持

**User Story:** As a user, I want the tool to work consistently on both macOS and Windows, so that I can use it on any of my computers.

#### Acceptance Criteria

1. THE Cleanup_CLI SHALL detect the current operating system at runtime
2. WHEN running on macOS, THE System_Cleaner SHALL use macOS-specific junk file locations
3. WHEN running on Windows, THE System_Cleaner SHALL use Windows-specific junk file locations
4. WHEN displaying paths, THE Cleanup_CLI SHALL use the appropriate path separator for the current OS
5. WHEN handling file permissions, THE Cleanup_CLI SHALL use OS-appropriate permission checks
6. THE Directory_Visualizer SHALL use Unicode box-drawing characters with fallback for terminals that don't support them
7. WHEN running on Windows, THE Cleanup_CLI SHALL handle both PowerShell and CMD environments
