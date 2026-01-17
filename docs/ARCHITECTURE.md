# Cleanup CLI - 架构设计文档

## 1. 系统概述

Cleanup CLI 是一个智能文件整理命令行工具，通过本地 AI 模型（Ollama/OpenAI）实现文件的智能分类、重命名和归档。

### 1.1 核心特性

- **AI 驱动的智能重命名**：自动识别无意义文件名，基于内容生成有意义的名称
- **文档场景分类**：智能识别文档类型（简历、面试、会议、报告等）
- **规则引擎**：灵活的文件整理规则配置
- **事务管理**：所有操作可撤销，支持回滚
- **系统清理**：扫描和清理系统垃圾文件
- **可视化对比**：清晰展示文件整理前后的目录结构差异

### 1.2 技术栈

- **语言**：Go 1.21+
- **AI 服务**：Ollama (本地) / OpenAI (云端)
- **配置管理**：Viper
- **CLI 框架**：Cobra
- **测试框架**：Testify + Property-based Testing (rapid)

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Scan    │  │ Organize │  │   Junk   │  │  Shell   │   │
│  │ Command  │  │ Command  │  │ Command  │  │Interactive│   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                     Business Logic Layer                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │Organizer │  │ Analyzer │  │  Cleaner │  │  Rules   │   │
│  │          │  │          │  │          │  │  Engine  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                 │
│  │Transaction│  │Visualizer│  │  Output  │                 │
│  │ Manager  │  │          │  │          │                 │
│  └──────────┘  └──────────┘  └──────────┘                 │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   AI     │  │  Config  │  │   File   │  │ Template │   │
│  │ Client   │  │ Manager  │  │  System  │  │  Engine  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 模块职责

#### 2.2.1 CLI Layer (cmd/cleanup)

**职责**：

- 命令行参数解析
- 用户交互界面
- 命令路由和调度

**主要组件**：

- `main.go`：应用入口，命令定义
- 命令处理器：scan, organize, junk, undo, history

#### 2.2.2 Business Logic Layer

##### Organizer (internal/organizer)

**职责**：

- 文件整理策略执行
- 文件移动和重命名操作
- 冲突处理
- 批量操作管理

**核心接口**：

```go
type Organizer interface {
    Organize(ctx context.Context, files []*FileMetadata, strategy *OrganizeStrategy) (*OrganizePlan, error)
    ExecutePlan(ctx context.Context, plan *OrganizePlan, strategy *OrganizeStrategy) (*BatchResult, error)
    Rename(ctx context.Context, source, newName string, opts *RenameOptions) (*OperationResult, error)
    Move(ctx context.Context, source, targetDir string, opts *MoveOptions) (*OperationResult, error)
}
```

##### Analyzer (internal/analyzer)

**职责**：

- 文件元数据提取
- 文件类型检测
- 文件名质量评估
- 目录扫描

**核心接口**：

```go
type Analyzer interface {
    Analyze(ctx context.Context, path string) (*FileMetadata, error)
    AnalyzeDirectory(ctx context.Context, path string, opts *ScanOptions) ([]*FileMetadata, error)
    DetectType(path string) (string, error)
    AssessFileNameQuality(filename string) FileNameQuality
}
```

**文件名质量评估**：

- `FileNameGood`：清晰有意义的文件名
- `FileNameGeneric`：通用名称，不够具体
- `FileNameMeaningless`：无意义的名称（需要 AI 重命名）

##### Rules Engine (internal/rules)

**职责**：

- 规则加载和管理
- 规则匹配
- 规则优先级处理

**规则类型**：

- Extension：基于文件扩展名
- Pattern：基于文件名模式（glob/regex）
- Size：基于文件大小
- Date：基于修改日期
- Composite：组合条件（and/or）

##### Transaction Manager (internal/transaction)

**职责**：

- 操作事务管理
- 操作日志记录
- 回滚支持
- 历史查询

**事务状态**：

- `Pending`：待提交
- `Committed`：已提交
- `Rolledback`：已回滚

##### Cleaner (internal/cleaner)

**职责**：

- 系统垃圾文件扫描
- 文件分类（缓存、日志、临时文件等）
- 安全删除（移至回收站）
- 交互式确认

**垃圾分类**：

- Cache：缓存文件
- Logs：日志文件
- Temp：临时文件
- Trash：回收站
- Browser：浏览器缓存
- Developer：开发工具缓存

##### Visualizer (internal/visualizer)

**职责**：

- 目录树可视化
- 文件变更对比
- 差异渲染

**功能**：

- 树形结构展示
- 前后对比
- 变更统计

#### 2.2.3 Infrastructure Layer

##### AI Client (internal/ai, internal/ollama)

**职责**：

- AI 服务抽象
- 文件名建议生成
- 文档场景分类

**支持的 AI 服务**：

- Ollama（本地）
- OpenAI（云端）

**核心接口**：

```go
type Client interface {
    CheckHealth(ctx context.Context) error
    Analyze(ctx context.Context, prompt string, context string) (*AnalysisResult, error)
    SuggestName(ctx context.Context, file *FileMetadata) ([]string, error)
    SuggestCategory(ctx context.Context, file *FileMetadata) ([]string, error)
}
```

##### Config Manager (internal/config)

**职责**：

- 配置文件加载
- 配置验证
- 默认值管理

**配置项**：

- AI 服务配置
- 规则定义
- 排除规则
- 清理器配置

##### Template Engine (pkg/template)

**职责**：

- 路径模板展开
- 占位符替换

**支持的占位符**：

- `{year}`：年份
- `{month}`：月份
- `{day}`：日期
- `{ext}`：文件扩展名
- `{category}`：文档场景分类

## 3. 数据流

### 3.1 文件整理流程

```
1. 扫描目录
   ↓
2. 文件分析（元数据提取）
   ↓
3. 文件名质量评估
   ↓
4. [如需要] AI 重命名
   ↓
5. [如需要] 文档场景分析
   ↓
6. 规则匹配
   ↓
7. 生成操作计划
   ↓
8. 用户确认（可选）
   ↓
9. 执行操作（并发）
   ↓
10. 记录事务
   ↓
11. 展示结果
```

### 3.2 系统清理流程

```
1. 扫描垃圾位置
   ↓
2. 文件分类
   ↓
3. 重要性评估
   ↓
4. 生成清理列表
   ↓
5. [交互模式] 用户确认
   ↓
6. 执行清理（移至回收站或永久删除）
   ↓
7. 记录事务
   ↓
8. 展示结果
```

## 4. 关键设计决策

### 4.1 并发处理

- 使用 goroutine 池限制并发数（默认 4）
- 使用 semaphore 控制并发访问
- 使用 sync.Mutex 保护共享状态

### 4.2 错误处理

- 操作级别的错误不中断整个流程
- 收集所有错误并在最后报告
- 支持部分成功的场景

### 4.3 事务管理

- 每个操作记录源路径、目标路径和备份路径
- 回滚时按相反顺序执行
- 事务日志持久化到 JSON 文件

### 4.4 AI 集成

- 抽象 AI 客户端接口，支持多种 AI 服务
- 仅在必要时调用 AI（文件名质量差或需要场景分类）
- 超时控制和错误降级

### 4.5 配置管理

- 使用 Viper 支持多种配置格式
- 配置文件优先级：命令行参数 > 配置文件 > 默认值
- 首次运行时启动设置向导

## 5. 扩展性设计

### 5.1 插件化规则引擎

规则引擎支持多种条件类型，易于扩展新的条件类型：

- 实现新的 `matchXXX` 方法
- 在 `matchesCondition` 中注册

### 5.2 多 AI 服务支持

通过 `ai.Client` 接口抽象，支持添加新的 AI 服务：

- 实现 `Client` 接口
- 在配置中添加新的 provider

### 5.3 自定义清理位置

支持通过配置文件添加自定义垃圾文件位置和重要文件模式。

## 6. 性能优化

### 6.1 文件扫描优化

- 使用 `filepath.Walk` 高效遍历
- 支持排除目录，减少扫描范围
- 并发分析文件元数据

### 6.2 AI 调用优化

- 仅对需要的文件调用 AI
- 批量处理时复用 AI 连接
- 设置合理的超时时间

### 6.3 内存管理

- 流式处理大目录
- 避免一次性加载所有文件到内存
- 及时释放不再使用的资源

## 7. 安全性考虑

### 7.1 文件操作安全

- 所有删除操作默认移至回收站
- 支持事务回滚
- 操作前验证路径合法性

### 7.2 权限处理

- 优雅处理权限错误
- 跳过无权访问的文件
- 记录跳过的文件供用户查看

### 7.3 数据隐私

- 本地 AI 模型（Ollama）保护隐私
- 不上传文件内容到云端（除非使用 OpenAI）
- 配置文件中的敏感信息（API Key）需要用户自行保护

## 8. 测试策略

### 8.1 单元测试

- 每个模块都有对应的测试文件
- 使用 testify 进行断言
- Mock 外部依赖（AI 客户端、文件系统）

### 8.2 集成测试

- `integration_test` 目录包含端到端测试
- 测试完整的文件整理流程
- 验证事务回滚功能

### 8.3 属性测试

- 使用 rapid 进行属性测试
- 测试事务管理的正确性
- 验证并发操作的安全性

## 9. 部署和分发

### 9.1 构建

- 使用 Makefile 管理构建流程
- 支持多平台编译
- 版本号在构建时注入

### 9.2 安装

- 提供安装脚本（install.sh）
- 支持 Homebrew 安装（本地 formula）
- 支持 macOS pkg 安装包

### 9.3 配置

- 首次运行启动设置向导
- 配置文件位于 `~/.cleanuprc.yaml`
- 事务日志位于 `~/.cleanup/transactions.json`

## 10. 未来规划

### 10.1 功能增强

- [ ] 支持更多文档场景分类
- [ ] 添加文件去重功能
- [ ] 支持云存储集成（S3、Google Drive）
- [ ] 添加定时任务支持

### 10.2 性能优化

- [ ] 实现增量扫描
- [ ] 添加文件索引缓存
- [ ] 优化大文件处理

### 10.3 用户体验

- [ ] 添加 Web UI
- [ ] 改进交互式界面（使用 Bubble Tea）
- [ ] 添加进度条和实时反馈

### 10.4 AI 能力

- [ ] 支持更多 AI 模型
- [ ] 添加文件内容摘要
- [ ] 智能标签生成
