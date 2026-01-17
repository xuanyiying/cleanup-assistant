# internal - 内部模块

## 概述

`internal` 目录包含 Cleanup CLI 的所有内部实现模块，这些模块不对外暴露，仅供项目内部使用。

## 模块结构

```
internal/
├── ai/              # AI 客户端抽象层
├── analyzer/        # 文件分析器
├── cleaner/         # 系统清理器
├── config/          # 配置管理
├── ollama/          # Ollama 客户端实现
├── organizer/       # 文件整理器
├── output/          # 输出格式化
├── rules/           # 规则引擎
├── setup/           # 首次运行设置向导
├── shell/           # 交互式 Shell
├── transaction/     # 事务管理
└── visualizer/      # 可视化工具
```

## 模块说明

### ai/ - AI 客户端抽象

定义 AI 客户端接口，支持多种 AI 服务（Ollama、OpenAI）。

**核心接口**：

```go
type Client interface {
    CheckHealth(ctx context.Context) error
    Analyze(ctx context.Context, prompt string, context string) (*AnalysisResult, error)
    SuggestName(ctx context.Context, file *FileMetadata) ([]string, error)
    SuggestCategory(ctx context.Context, file *FileMetadata) ([]string, error)
}
```

**子模块**：

- `openai/`：OpenAI 客户端实现

### analyzer/ - 文件分析器

负责文件元数据提取、类型检测、文件名质量评估。

**核心功能**：

- 文件元数据提取（大小、类型、修改时间等）
- MIME 类型检测（magic bytes + 扩展名）
- 文件名质量评估（good/generic/meaningless）
- 目录递归扫描
- 排除规则支持

### cleaner/ - 系统清理器

扫描和清理系统垃圾文件（缓存、日志、临时文件）。

**核心功能**：

- 垃圾文件扫描
- 文件分类（cache/logs/temp/trash）
- 重要文件识别
- 交互式确认
- 安全删除（移至回收站）

**子模块**：

- `scanner.go`：垃圾文件扫描器
- `classifier.go`：文件分类器
- `prompt.go`：交互式提示
- `platform.go`：平台特定逻辑

### config/ - 配置管理

使用 Viper 管理配置文件，支持 YAML 格式。

**配置项**：

- AI 服务配置（Ollama/OpenAI）
- 规则定义
- 排除规则
- 清理器配置
- 默认策略

### ollama/ - Ollama 客户端

Ollama 本地 AI 服务的客户端实现。

**核心功能**：

- 健康检查
- 文本生成
- 文件名建议
- 文档场景分类

### organizer/ - 文件整理器

核心业务逻辑，负责文件的移动、重命名和整理。

**核心功能**：

- 文件移动
- 文件重命名
- 冲突处理（skip/suffix/overwrite）
- 批量操作
- 并发执行
- 事务支持

### output/ - 输出格式化

提供格式化输出和样式支持。

**核心功能**：

- 控制台输出
- 颜色和样式
- 进度显示

### rules/ - 规则引擎

基于规则的文件匹配和操作。

**规则类型**：

- Extension：文件扩展名匹配
- Pattern：文件名模式匹配（glob/regex）
- Size：文件大小匹配
- Date：修改日期匹配
- Composite：组合条件（and/or）

### setup/ - 设置向导

首次运行时的交互式设置向导。

**功能**：

- AI 服务配置
- 规则配置
- 生成配置文件

### shell/ - 交互式 Shell

提供交互式命令行界面。

**功能**：

- 自然语言命令解析
- 命令历史
- 自动补全

### transaction/ - 事务管理

提供操作的事务支持和回滚功能。

**核心功能**：

- 事务开始/提交/回滚
- 操作日志记录
- 历史查询
- 持久化存储

### visualizer/ - 可视化工具

提供目录树和文件变更的可视化。

**核心功能**：

- 目录树渲染
- 文件变更对比
- 差异可视化

## 依赖关系

```
cmd/cleanup
    ↓
internal/organizer ← internal/analyzer
    ↓                    ↓
internal/rules      internal/ai
    ↓                    ↓
internal/transaction  internal/ollama
    ↓
internal/config
```

## 设计原则

1. **接口驱动**：核心组件都定义接口，便于测试和扩展
2. **依赖注入**：通过构造函数注入依赖
3. **单一职责**：每个模块职责明确
4. **错误处理**：使用 error wrapping 提供上下文
5. **并发安全**：使用 mutex 保护共享状态
6. **Context 支持**：长时间操作支持取消和超时

## 测试

每个模块都有对应的测试文件：

```bash
# 运行所有测试
go test ./internal/...

# 运行特定模块测试
go test ./internal/analyzer -v

# 运行测试并显示覆盖率
go test ./internal/... -cover
```

## 扩展指南

### 添加新模块

1. 在 `internal/` 下创建新目录
2. 定义接口和实现
3. 添加测试文件
4. 更新此 README

### 实现新的 AI 客户端

1. 在 `internal/ai/` 下创建新目录
2. 实现 `ai.Client` 接口
3. 在配置中添加新的 provider
4. 在 `cmd/cleanup/main.go` 中注册

### 添加新的规则类型

1. 在 `internal/rules/engine.go` 中添加新的 `matchXXX` 方法
2. 在 `matchesCondition` 中注册新类型
3. 更新配置文档
4. 添加测试

## 最佳实践

1. **使用 context**：所有长时间运行的操作都应接受 context
2. **错误包装**：使用 `fmt.Errorf` 和 `%w` 包装错误
3. **日志记录**：使用统一的日志接口
4. **资源清理**：使用 defer 确保资源释放
5. **并发控制**：使用 semaphore 限制并发数
6. **输入验证**：验证所有外部输入

## 常见问题

### Q: 如何添加新的文件类型支持？

A: 在 `internal/analyzer/analyzer.go` 的 `detectByMagicBytes` 方法中添加新的 magic bytes 检测。

### Q: 如何自定义规则？

A: 编辑 `~/.cleanuprc.yaml` 配置文件，添加新的规则定义。

### Q: 如何扩展清理器支持新的垃圾位置？

A: 在配置文件的 `cleaner.junkLocations` 中添加新路径，或在代码中修改 `internal/cleaner/scanner.go` 的默认位置。

## 参考资料

- [架构设计文档](../docs/ARCHITECTURE.md)
- [代码分析报告](../docs/CODE_ANALYSIS.md)
- [API 文档](https://pkg.go.dev/github.com/xuanyiying/cleanup-cli)
