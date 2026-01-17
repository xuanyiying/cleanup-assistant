# cmd/cleanup - CLI 入口

## 概述

这是 Cleanup CLI 的命令行入口，负责解析命令行参数、路由命令和协调各个模块。

## 文件结构

```
cmd/cleanup/
├── main.go       # 主入口文件，命令定义和路由
└── main_test.go  # 主入口测试
```

## 主要功能

### 1. 命令定义

使用 Cobra 框架定义以下命令：

- `cleanup`：进入交互模式
- `cleanup scan [path]`：扫描目录
- `cleanup organize [path]`：整理文件
- `cleanup junk scan`：扫描垃圾文件
- `cleanup junk clean`：清理垃圾文件
- `cleanup undo [txn-id]`：撤销操作
- `cleanup history`：查看历史
- `cleanup version`：查看版本

### 2. 全局标志

- `--config`：配置文件路径
- `--dry-run`：预览模式
- `--model`：AI 模型名称
- `--exclude-ext`：排除的文件扩展名
- `--exclude-pattern`：排除的文件模式
- `--exclude-dir`：排除的目录

### 3. 依赖初始化

在 `init()` 函数中初始化全局依赖：

```go
configMgr = config.NewManager(defaultConfigPath)
txnMgr = transaction.NewManager(txnLogPath)
fileAnalyzer = analyzer.NewAnalyzer()
ruleEngine = rules.NewEngine()
fileOrganizer = organizer.NewOrganizerWithDeps(txnMgr, ruleEngine, fileAnalyzer)
systemCleaner = cleaner.NewSystemCleaner(txnMgr)
aiClient = ollama.NewClient(&cfg.Ollama)
```

### 4. 首次运行设置

如果配置文件不存在，自动启动设置向导：

```go
if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
    if err := setup.RunSetup(configMgr); err != nil {
        // 使用默认配置继续
    }
}
```

## 命令实现

### scan 命令

扫描目录并显示文件信息：

```go
files, err := fileAnalyzer.AnalyzeDirectory(ctx, absPath, scanOpts)
for _, file := range files {
    fmt.Printf("  - %s (%s, %d bytes)\n", file.Name, file.MimeType, file.Size)
}
```

### organize 命令

整理文件的完整流程：

1. 扫描目录
2. 捕获前置状态（用于 diff）
3. 生成整理计划
4. 显示计划摘要
5. 执行计划（如果不是 dry-run）
6. 捕获后置状态并显示 diff
7. 显示执行结果

### junk 命令

系统清理命令组：

- `junk scan`：扫描垃圾文件
- `junk clean`：清理垃圾文件

支持按类别过滤：`--category cache|logs|temp|trash|all`

### undo 命令

撤销操作：

```go
// 如果没有指定事务 ID，撤销最后一次操作
if len(args) == 0 {
    history, _ := txnMgr.GetHistory(1)
    txnID = history[0].ID
}
txnMgr.Undo(txnID)
```

### history 命令

查看事务历史：

```go
history, err := txnMgr.GetHistory(limit)
for _, txn := range history {
    fmt.Printf("  ID: %s\n", txn.ID)
    fmt.Printf("    Time: %s\n", txn.Timestamp)
    fmt.Printf("    Status: %s\n", txn.Status)
}
```

## 辅助函数

### buildScanOptions

构建扫描选项，合并命令行参数和配置文件：

```go
func buildScanOptions() *analyzer.ScanOptions {
    opts := &analyzer.ScanOptions{
        Recursive:     true,
        IncludeHidden: false,
    }

    // 合并命令行参数
    opts.ExcludeExtensions = append(opts.ExcludeExtensions, excludeExtensions...)

    // 合并配置文件
    cfg, _ := configMgr.Load()
    opts.ExcludeExtensions = append(opts.ExcludeExtensions, cfg.Exclude.Extensions...)

    return opts
}
```

### interactiveMode

进入交互式 Shell：

```go
func interactiveMode() error {
    // 检查 AI 服务可用性
    if err := aiClient.CheckHealth(ctx); err != nil {
        return err
    }

    // 启动交互式 Shell
    interactiveShell := shell.NewInteractiveShell(...)
    return interactiveShell.Start()
}
```

## 错误处理

所有命令都返回错误，由 Cobra 框架统一处理：

```go
if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
}
```

## 输出格式

使用 Unicode 字符和颜色增强输出：

```
╔════════════════════════════════════════╗
║       Organization Plan Summary        ║
╚════════════════════════════════════════╝
  Total files:      10
  Total operations: 8
  Moves:            5
  Renames:          3
  Skipped:          2
```

## 使用示例

```bash
# 扫描当前目录
cleanup scan

# 整理下载目录（预览模式）
cleanup organize --dry-run ~/Downloads

# 整理并排除特定文件
cleanup organize ~/Documents \
  --exclude-ext log,tmp \
  --exclude-dir .git,node_modules

# 清理系统垃圾
cleanup junk scan
cleanup junk clean

# 撤销最后一次操作
cleanup undo

# 查看历史
cleanup history
```

## 扩展指南

### 添加新命令

1. 定义命令变量：

```go
var newCmd = &cobra.Command{
    Use:   "new",
    Short: "New command description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 实现命令逻辑
        return nil
    },
}
```

2. 在 `init()` 中注册：

```go
rootCmd.AddCommand(newCmd)
```

### 添加新标志

```go
newCmd.Flags().StringVar(&flagVar, "flag-name", "default", "description")
```

## 测试

运行测试：

```bash
go test ./cmd/cleanup -v
```

## 依赖

- `github.com/spf13/cobra`：CLI 框架
- `github.com/spf13/viper`：配置管理
- 内部模块：analyzer, organizer, cleaner, rules, transaction, ai, config
