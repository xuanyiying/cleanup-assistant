# Cleanup CLI - 优化计划

## 1. Bug 修复计划

### 1.1 高优先级 Bug

#### Bug #1: 文件名冲突处理改进

**问题**：当前使用计数器生成唯一文件名，可能在极端情况下失败。

**位置**：`internal/organizer/organizer.go:generateUniquePath()`

**修复方案**：

```go
func (o *Organizer) generateUniquePath(targetPath string) string {
    dir := filepath.Dir(targetPath)
    fileName := filepath.Base(targetPath)
    ext := filepath.Ext(fileName)
    baseName := strings.TrimSuffix(fileName, ext)

    // 使用时间戳 + 随机数确保唯一性
    timestamp := time.Now().UnixNano()
    random := rand.Intn(10000)
    newName := fmt.Sprintf("%s_%d_%d%s", baseName, timestamp, random, ext)
    return filepath.Join(dir, newName)
}
```

**预计工作量**：2 小时

#### Bug #2: 事务回滚容错改进

**问题**：回滚过程中如果某个操作失败，后续操作不会执行。

**位置**：`internal/transaction/manager.go:Rollback()`

**修复方案**：

```go
func (m *Manager) Rollback(tx *Transaction) error {
    var errors []error

    // 尝试回滚所有操作，收集错误
    for i := len(tx.Operations) - 1; i >= 0; i-- {
        if err := m.rollbackOperation(tx.Operations[i]); err != nil {
            errors = append(errors, err)
        }
    }

    // 如果有错误，返回聚合错误
    if len(errors) > 0 {
        return fmt.Errorf("rollback partially failed: %v", errors)
    }

    return nil
}
```

**预计工作量**：4 小时

#### Bug #3: 路径遍历漏洞修复

**问题**：文件名可能包含 `../` 导致路径遍历。

**位置**：`internal/organizer/organizer.go:Move()`

**修复方案**：

```go
func (o *Organizer) Move(ctx context.Context, source, targetDir string, opts *MoveOptions) (*OperationResult, error) {
    // 清理和验证路径
    targetDir = filepath.Clean(targetDir)
    fileName = filepath.Clean(filepath.Base(source))

    // 确保文件名不包含路径分隔符
    if strings.Contains(fileName, string(filepath.Separator)) {
        return nil, fmt.Errorf("invalid filename: %s", fileName)
    }

    targetPath := filepath.Join(targetDir, fileName)

    // 验证最终路径在目标目录内
    if !strings.HasPrefix(targetPath, targetDir) {
        return nil, fmt.Errorf("path traversal detected")
    }

    // ... 继续执行
}
```

**预计工作量**：3 小时

### 1.2 中优先级 Bug

#### Bug #4: 并发文件操作冲突

**问题**：多个 goroutine 可能同时操作同一个文件。

**修复方案**：

- 在执行前检查源文件是否存在
- 使用文件路径作为锁的 key
- 实现简单的文件锁机制

**预计工作量**：6 小时

## 2. 性能优化计划

### 2.1 文件扫描优化

**当前问题**：

- 串行扫描文件
- 每个文件都计算哈希
- 大目录扫描慢

**优化方案**：

1. **并发扫描**：

```go
func (fa *FileAnalyzer) AnalyzeDirectory(ctx context.Context, path string, opts *ScanOptions) ([]*FileMetadata, error) {
    // 使用 worker pool 并发分析文件
    workers := 4
    fileChan := make(chan string, 100)
    resultChan := make(chan *FileMetadata, 100)

    // 启动 workers
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for filePath := range fileChan {
                if metadata, err := fa.Analyze(ctx, filePath); err == nil {
                    resultChan <- metadata
                }
            }
        }()
    }

    // ... 扫描文件并发送到 fileChan
}
```

2. **可选哈希计算**：

```go
type ScanOptions struct {
    // ... 现有字段
    CalculateHash bool // 是否计算文件哈希
}
```

3. **增量扫描**：

- 缓存上次扫描结果
- 只扫描修改过的文件

**预计性能提升**：3-5倍
**预计工作量**：8 小时

### 2.2 AI 调用优化

**当前问题**：

- AI 调用串行执行
- 没有批量处理
- 没有缓存

**优化方案**：

1. **并发 AI 调用**：

```go
func (o *Organizer) Organize(ctx context.Context, files []*FileMetadata, strategy *OrganizeStrategy) (*OrganizePlan, error) {
    // 收集需要 AI 处理的文件
    var aiFiles []*FileMetadata
    for _, file := range files {
        if file.NeedsSmarterName {
            aiFiles = append(aiFiles, file)
        }
    }

    // 并发调用 AI
    results := o.batchAIProcess(ctx, aiFiles, 4) // 4 个并发

    // ... 处理结果
}
```

2. **AI 响应缓存**：

```go
type AICache struct {
    cache map[string]string
    mu    sync.RWMutex
}

func (c *AICache) Get(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.cache[key]
    return val, ok
}
```

3. **批量请求**：

- 实现批量 AI 请求接口
- 减少网络往返

**预计性能提升**：5-10倍
**预计工作量**：12 小时

### 2.3 内存优化

**当前问题**：

- 所有文件元数据加载到内存
- 大目录可能导致 OOM

**优化方案**：

1. **流式处理**：

```go
type FileStream struct {
    files chan *FileMetadata
    done  chan struct{}
}

func (fa *FileAnalyzer) StreamDirectory(ctx context.Context, path string) *FileStream {
    stream := &FileStream{
        files: make(chan *FileMetadata, 100),
        done:  make(chan struct{}),
    }

    go func() {
        defer close(stream.files)
        defer close(stream.done)

        filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
            if !info.IsDir() {
                metadata, _ := fa.Analyze(ctx, filePath)
                stream.files <- metadata
            }
            return nil
        })
    }()

    return stream
}
```

2. **分批处理**：

- 每次处理固定数量的文件
- 处理完后释放内存

3. **内存限制**：

- 监控内存使用
- 达到阈值时暂停扫描

**预计内存节省**：50-70%
**预计工作量**：10 小时

## 3. 代码质量改进

### 3.1 减少代码重复

**问题区域**：

- 错误处理代码重复
- 文件操作代码重复

**改进方案**：

1. **提取通用错误处理**：

```go
// pkg/errors/errors.go
func WrapError(err error, format string, args ...interface{}) error {
    if err == nil {
        return nil
    }
    msg := fmt.Sprintf(format, args...)
    return fmt.Errorf("%s: %w", msg, err)
}
```

2. **提取文件操作工具**：

```go
// pkg/fileutil/fileutil.go
func SafeRename(src, dst string) error {
    // 统一的重命名逻辑
}

func SafeMove(src, dst string) error {
    // 统一的移动逻辑
}
```

**预计工作量**：6 小时

### 3.2 消除魔法数字

**改进方案**：

```go
// internal/organizer/constants.go
const (
    MaxConflictRetries = 1000
    DefaultConcurrency = 4
    MaxPreviewLength   = 500
    MaxDisplayOperations = 20
)
```

**预计工作量**：2 小时

### 3.3 添加输入验证

**改进方案**：

```go
// pkg/validator/validator.go
func ValidateFilename(name string) error {
    if name == "" {
        return errors.New("filename cannot be empty")
    }

    // 检查非法字符
    invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
    for _, char := range invalidChars {
        if strings.Contains(name, char) {
            return fmt.Errorf("filename contains invalid character: %s", char)
        }
    }

    return nil
}

func ValidatePath(path string) error {
    // 路径验证逻辑
}
```

**预计工作量**：4 小时

## 4. 测试改进计划

### 4.1 提高测试覆盖率

**目标**：从 30% 提升到 70%+

**计划**：

1. **Analyzer 模块**（当前 45% → 目标 75%）
   - 添加边界条件测试
   - 添加错误场景测试
   - 预计工作量：4 小时

2. **Organizer 模块**（当前 25% → 目标 70%）
   - 添加并发测试
   - 添加冲突处理测试
   - 预计工作量：8 小时

3. **Cleaner 模块**（当前 30% → 目标 70%）
   - 添加平台特定测试
   - 添加交互式测试
   - 预计工作量：6 小时

4. **Rules 模块**（当前 40% → 目标 75%）
   - 添加复杂规则测试
   - 添加性能测试
   - 预计工作量：4 小时

### 4.2 添加集成测试

**计划**：

1. **端到端测试**：
   - 完整的文件整理流程
   - 事务回滚测试
   - 预计工作量：8 小时

2. **性能测试**：
   - 大文件测试
   - 大目录测试
   - 并发测试
   - 预计工作量：6 小时

### 4.3 添加基准测试

```go
func BenchmarkAnalyzeDirectory(b *testing.B) {
    analyzer := NewAnalyzer()
    for i := 0; i < b.N; i++ {
        analyzer.AnalyzeDirectory(context.Background(), testDir, nil)
    }
}
```

**预计工作量**：4 小时

## 5. 文档改进计划

### 5.1 API 文档

**任务**：

- 为所有公共函数添加 godoc 注释
- 添加使用示例
- 添加参数说明

**预计工作量**：8 小时

### 5.2 用户文档

**任务**：

- 完善 README
- 添加常见问题解答
- 添加故障排除指南
- 添加最佳实践

**预计工作量**：6 小时

### 5.3 开发者文档

**任务**：

- 添加贡献指南
- 添加开发环境设置指南
- 添加发布流程文档

**预计工作量**：4 小时

## 6. 功能完善计划

### 6.1 缺失功能

1. **文件去重**：
   - 基于哈希的去重
   - 交互式选择保留哪个
   - 预计工作量：12 小时

2. **定时任务**：
   - 定期自动整理
   - Cron 表达式支持
   - 预计工作量：8 小时

3. **云存储集成**：
   - S3 支持
   - Google Drive 支持
   - 预计工作量：20 小时

4. **Web UI**：
   - 基于 Web 的管理界面
   - 实时进度显示
   - 预计工作量：40 小时

### 6.2 用户体验改进

1. **进度条**：
   - 实时进度显示
   - 预计剩余时间
   - 预计工作量：4 小时

2. **更好的交互式界面**：
   - 使用 Bubble Tea 框架
   - 更丰富的 TUI
   - 预计工作量：16 小时

3. **配置向导改进**：
   - 更友好的交互
   - 配置验证
   - 预计工作量：6 小时

## 7. 实施时间表

### 第一阶段（1-2 周）：紧急修复

- [x] Bug #1: 文件名冲突处理（2h）✅ 已完成
- [x] Bug #2: 事务回滚容错（4h）✅ 已完成
- [x] Bug #3: 路径遍历漏洞（3h）✅ 已完成
- [x] 添加输入验证（4h）✅ 已完成
- [x] 消除魔法数字（2h）✅ 已完成

**总计**：15 小时 ✅ **已完成**

### 第二阶段（2-4 周）：性能优化

- [x] 文件扫描优化（8h）✅ 已完成
- [x] AI 调用优化（12h）✅ 已完成
- [ ] 内存优化（10h）
- [x] Bug #4: 并发冲突（6h）✅ 已完成

**总计**：36 小时 **进度**: 26/36 小时 (72%)

### 第三阶段（4-6 周）：质量提升

- [x] 减少代码重复（6h）✅ 已完成
- [x] 提高测试覆盖率（22h）✅ 已完成 (已达85%+)
- [ ] 添加集成测试（8h）
- [ ] 添加基准测试（4h）

**总计**：40 小时 **进度**: 28/40 小时 (70%)

### 第四阶段（6-8 周）：文档完善

- [x] API 文档（8h）✅ 已完成
- [x] 用户文档（6h）✅ 已完成
- [x] 开发者文档（4h）✅ 已完成

**总计**：18 小时 ✅ **已完成**

### 第五阶段（8-12 周）：功能扩展

- [x] 文件去重（12h）✅ 已完成
- [x] 定时任务（8h）✅ 已完成
- [x] 进度条（4h）✅ 已完成
- [x] 配置向导改进（6h）✅ 已完成

**总计**：30 小时 ✅ **已完成**

### 长期计划（3-6 个月）

- [ ] 云存储集成（20h）
- [ ] Web UI（40h）
- [ ] 更好的交互式界面（16h）

**总计**：76 小时

## 8. 优先级矩阵

| 任务             | 优先级 | 影响 | 工作量 | 状态    |
| ---------------- | ------ | ---- | ------ | ------- |
| Bug #1-3 修复    | 高     | 高   | 9h     | ✅ 完成 |
| 输入验证         | 高     | 高   | 4h     | ✅ 完成 |
| 文件扫描优化     | 高     | 高   | 8h     | ✅ 完成 |
| AI 调用优化      | 高     | 高   | 12h    | ✅ 完成 |
| Bug #4: 并发冲突 | 高     | 高   | 6h     | ✅ 完成 |
| 代码重复消除     | 中     | 中   | 6h     | ✅ 完成 |
| 测试覆盖率提升   | 中     | 高   | 22h    | ✅ 完成 |
| API 文档         | 中     | 中   | 8h     | ✅ 完成 |
| 用户文档         | 中     | 中   | 6h     | ✅ 完成 |
| 开发者文档       | 中     | 中   | 4h     | ✅ 完成 |
| 内存优化         | 中     | 中   | 10h    | 待开始  |
| 文件去重         | 中     | 中   | 12h    | ✅ 完成 |
| 定时任务         | 中     | 中   | 8h     | ✅ 完成 |
| 进度条           | 中     | 中   | 4h     | ✅ 完成 |
| 配置向导改进     | 中     | 中   | 6h     | ✅ 完成 |
| Web UI           | 低     | 低   | 40h    | 待开始  |

## 9. 成功指标

### 9.1 性能指标

- 文件扫描速度提升 3-5 倍
- AI 调用速度提升 5-10 倍
- 内存使用减少 50-70%

### 9.2 质量指标

- 测试覆盖率达到 70%+
- 零已知安全漏洞
- 零高优先级 Bug

### 9.3 用户体验指标

- 文档完整度 90%+
- 用户满意度 4.5/5+
- 问题响应时间 < 24h

## 10. 风险评估

### 10.1 技术风险

- **并发优化可能引入新 Bug**：通过充分测试降低风险
- **性能优化可能影响稳定性**：分阶段实施，逐步验证
- **API 变更可能破坏兼容性**：遵循语义化版本控制

### 10.2 资源风险

- **开发时间不足**：优先实施高优先级任务
- **测试资源有限**：自动化测试，持续集成

### 10.3 缓解措施

- 分阶段实施，每个阶段都有明确的交付物
- 充分测试，确保每个变更都经过验证
- 保持向后兼容，避免破坏性变更
- 及时沟通，收集用户反馈

## 11. 总结

本优化计划涵盖了 Bug 修复、性能优化、代码质量改进、测试完善、文档补充和功能扩展等多个方面。

**总预计工作量**：约 215 小时（约 5-6 周全职工作）

**关键里程碑**：

- 第 2 周：完成紧急 Bug 修复
- 第 4 周：完成性能优化
- 第 6 周：完成质量提升
- 第 8 周：完成文档完善
- 第 12 周：完成功能扩展

通过系统化的优化，Cleanup CLI 将成为一个高质量、高性能、易用的文件整理工具。
