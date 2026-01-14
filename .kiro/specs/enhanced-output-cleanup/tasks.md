# Implementation Plan: Enhanced Output & System Cleanup

## Overview

本实现计划将新功能分为 6 个主要阶段：

1. 控制台输出基础模块
2. 目录可视化模块
3. 差异渲染模块
4. 垃圾扫描模块
5. 系统清理模块
6. 集成与 CLI 命令

## Tasks

- [x] 1. 实现控制台输出基础模块

  - [x] 1.1 创建 internal/output/style.go - 颜色样式模块
    - 实现 Color 常量和 Style 结构体
    - 实现 Styler 类型及 Red/Green/Yellow/Blue/Bold/Dim 方法
    - 实现 ANSI 转义码生成和禁用时的降级处理
    - _Requirements: 3.1, 3.7_
  - [x] 1.2 编写 style.go 的属性测试
    - **Property 7: Color Fallback**
    - **Validates: Requirements 3.1, 3.7**
  - [x] 1.3 创建 internal/output/console.go - 控制台输出模块
    - 实现 Console 类型和 NewConsole 构造函数
    - 实现 DetectColorSupport 检测终端颜色能力
    - 实现 Success/Error/Warning/Info 方法
    - 实现 Box/Table 格式化输出方法
    - _Requirements: 3.3, 3.4, 3.5, 3.6_
  - [x] 1.4 编写 console.go 的属性测试
    - **Property 6: Message Type Styling**
    - **Validates: Requirements 3.3, 3.4, 3.5**

- [x] 2. Checkpoint - 确保所有测试通过

  - 运行 `go test ./internal/output/...`
  - 如有问题请询问用户

- [x] 3. 实现目录可视化模块

  - [x] 3.1 创建 internal/visualizer/tree.go - 树形可视化
    - 实现 TreeNode 结构体和 TreeOptions 配置
    - 实现 TreeVisualizer 类型和 NewTreeVisualizer 构造函数
    - 实现 BuildTree 方法递归构建目录树
    - 实现 Render 方法生成树形字符串 (├── └── │)
    - 实现 Unicode/ASCII 分支字符切换
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_
  - [x] 3.2 编写 tree.go 的属性测试
    - **Property 1: Tree Rendering Structure Correctness**
    - **Validates: Requirements 1.1, 1.2**
  - [x] 3.3 编写 tree.go 的深度限制属性测试
    - **Property 2: Tree Depth Limiting**
    - **Validates: Requirements 1.5**
  - [x] 3.4 编写 tree.go 的 Unicode 降级属性测试
    - **Property 22: Unicode Fallback**
    - **Validates: Requirements 8.6**

- [x] 4. 实现差异渲染模块

  - [x] 4.1 创建 internal/visualizer/diff.go - 差异渲染
    - 实现 DiffType 常量和 DiffEntry/DiffResult 结构体
    - 实现 DiffRenderer 类型和 NewDiffRenderer 构造函数
    - 实现 CaptureState 方法捕获目录状态
    - 实现 Compare 方法对比两个状态
    - 实现 Render 方法渲染差异 (+绿/-红/→ 黄)
    - 实现 RenderSummary 方法显示变化统计
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7_
  - [x] 4.2 编写 diff.go 的状态捕获属性测试
    - **Property 3: Diff State Capture Round-Trip**
    - **Validates: Requirements 2.1, 2.2**
  - [x] 4.3 编写 diff.go 的高亮属性测试
    - **Property 4: Diff Highlighting Correctness**
    - **Validates: Requirements 2.3, 2.4, 2.5**
  - [x] 4.4 编写 diff.go 的统计准确性属性测试
    - **Property 5: Diff Summary Accuracy**
    - **Validates: Requirements 2.7**

- [x] 5. Checkpoint - 确保所有测试通过

  - 运行 `go test ./internal/visualizer/...`
  - 如有问题请询问用户

- [-] 6. 实现文件分类器模块

  - [x] 6.1 创建 internal/cleaner/classifier.go - 文件分类器
    - 实现 FileImportance 常量和 ImportantPattern 结构体
    - 实现 FileClassifier 类型和 NewFileClassifier 构造函数
    - 实现 GetDefaultPatterns 返回默认重要文件模式
    - 实现 Classify 方法判断文件重要性
    - 实现 IsImportant/IsUncertain 便捷方法
    - _Requirements: 6.1, 6.2, 6.3, 6.4_
  - [x] 6.2 编写 classifier.go 的属性测试
    - **Property 15: Important File Classification**
    - **Validates: Requirements 6.1**

- [x] 7. 实现交互式提示模块

  - [x] 7.1 创建 internal/cleaner/prompt.go - 交互式提示
    - 实现 PromptAction 常量和 FilePrompt 结构体
    - 实现 InteractivePrompt 类型和 NewInteractivePrompt 构造函数
    - 实现 Prompt 方法显示单文件确认 [Y/N/A/S/V]
    - 实现 ShowPreview 方法显示文件预览 (最多 500 字符)
    - 实现 allYes/skipAll 批量处理状态
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 7.8_
  - [x] 7.2 编写 prompt.go 的预览长度属性测试
    - **Property 17: File Preview Length**
    - **Validates: Requirements 7.4**
  - [x] 7.3 编写 prompt.go 的批量行为属性测试
    - **Property 18: All Yes Behavior**
    - **Property 19: Skip All Behavior**
    - **Validates: Requirements 7.5, 7.6**

- [x] 8. 实现垃圾扫描模块

  - [x] 8.1 创建 internal/cleaner/scanner.go - 垃圾扫描器
    - 实现 JunkCategory 常量和 JunkLocation/JunkFile 结构体
    - 实现 JunkScanner 类型和 NewJunkScanner 构造函数
    - 实现 GetDefaultLocations 返回平台特定垃圾位置
    - 实现 Scan 方法扫描垃圾文件
    - 实现 ScanCategory 方法按类别扫描
    - 集成 FileClassifier 排除重要文件
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_
  - [x] 8.2 编写 scanner.go 的平台检测属性测试
    - **Property 8: Platform-Specific Junk Detection**
    - **Validates: Requirements 4.1, 4.2**
  - [x] 8.3 编写 scanner.go 的大小计算属性测试
    - **Property 9: Junk Size Calculation**
    - **Validates: Requirements 4.3**
  - [x] 8.4 编写 scanner.go 的分类属性测试
    - **Property 10: Junk Categorization**
    - **Validates: Requirements 4.4**
  - [x] 8.5 编写 scanner.go 的重要文件排除属性测试
    - **Property 16: Important File Exclusion**
    - **Validates: Requirements 6.5**

- [x] 9. Checkpoint - 确保所有测试通过

  - 运行 `go test ./internal/cleaner/...`
  - 如有问题请询问用户

- [x] 10. 实现系统清理模块

  - [x] 10.1 创建 internal/cleaner/cleaner.go - 系统清理器
    - 实现 CleanOptions 和 CleanResult 结构体
    - 实现 SystemCleaner 类型和 NewSystemCleaner 构造函数
    - 实现 Preview 方法预览待清理文件
    - 实现 Clean 方法执行清理 (移至回收站)
    - 实现 --force 永久删除逻辑
    - 集成 InteractivePrompt 处理不确定文件
    - 集成 Transaction Manager 支持回滚
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8_
  - [x] 10.2 编写 cleaner.go 的回收站属性测试
    - **Property 11: Trash vs Permanent Delete**
    - **Validates: Requirements 5.3**
  - [x] 10.3 编写 cleaner.go 的强制删除属性测试
    - **Property 12: Force Delete Behavior**
    - **Validates: Requirements 5.4**
  - [x] 10.4 编写 cleaner.go 的空间计算属性测试
    - **Property 13: Space Freed Calculation**
    - **Validates: Requirements 5.5**
  - [x] 10.5 编写 cleaner.go 的错误恢复属性测试
    - **Property 14: Error Resilience**
    - **Validates: Requirements 5.6**

- [x] 11. 实现平台适配模块

  - [x] 11.1 创建 internal/cleaner/platform.go - 平台适配
    - 实现 GetPlatform 检测当前操作系统
    - 实现 ExpandPath 展开环境变量 (~, %TEMP% 等)
    - 实现 GetPathSeparator 返回平台路径分隔符
    - 实现 IsProtectedPath 检查受保护路径 (SIP 等)
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_
  - [x] 11.2 编写 platform.go 的路径分隔符属性测试
    - **Property 21: Platform Path Separator**
    - **Validates: Requirements 8.4**

- [x] 12. Checkpoint - 确保所有测试通过

  - 运行 `go test ./internal/cleaner/...`
  - 如有问题请询问用户

- [x] 13. 集成到 CLI 命令

  - [x] 13.1 更新 cmd/cleanup/main.go - 添加新命令
    - 添加 `cleanup junk scan` 命令扫描垃圾文件
    - 添加 `cleanup junk clean` 命令清理垃圾文件
    - 添加 `--category` 参数按类别过滤
    - 添加 `--force` 参数永久删除
    - 添加 `--dry-run` 参数预览模式
    - _Requirements: 5.1, 5.2, 5.7_
  - [x] 13.2 更新 organize 命令集成差异显示
    - 在 organize 操作前后捕获目录状态
    - 操作完成后显示树形对比
    - 显示变化统计摘要
    - _Requirements: 2.1, 2.2, 2.7_
  - [x] 13.3 更新配置文件支持
    - 在 config.go 中添加 output/cleaner 配置结构
    - 支持自定义垃圾位置和重要文件模式
    - _Requirements: 4.5, 6.4_

- [x] 14. 编写集成测试

  - [x] 14.1 创建 integration_test/cleaner_test.go
    - 测试完整的扫描-清理流程
    - 测试跨平台路径处理
    - 测试事务回滚功能
    - _Requirements: 5.3, 5.6, 7.7_

- [x] 15. Final Checkpoint - 确保所有测试通过
  - 运行 `go test ./...`
  - 如有问题请询问用户

## Notes

- 所有任务都是必须完成的，包括测试任务
- 每个任务引用具体的需求条款以确保可追溯性
- Checkpoint 任务确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
