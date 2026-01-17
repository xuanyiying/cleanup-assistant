# pkg - 公共包

## 概述

`pkg` 目录包含可以被外部项目使用的公共包。这些包提供通用功能，不依赖项目的内部实现。

## 包结构

```
pkg/
└── template/        # 模板引擎
    ├── template.go
    └── template_test.go
```

## template - 模板引擎

路径模板展开工具，支持占位符替换。

### 功能

- 路径模板展开
- 占位符替换
- 嵌套路径处理

### 支持的占位符

| 占位符       | 说明         | 示例                       |
| ------------ | ------------ | -------------------------- |
| `{year}`     | 年份（4位）  | 2024                       |
| `{month}`    | 月份（2位）  | 01                         |
| `{day}`      | 日期（2位）  | 15                         |
| `{ext}`      | 文件扩展名   | pdf                        |
| `{category}` | 文档场景分类 | resume, interview, meeting |

### 使用示例

```go
package main

import (
    "fmt"
    "github.com/xuanyiying/cleanup-cli/pkg/template"
)

func main() {
    // 创建模板展开器
    placeholders := map[string]string{
        "year":     "2024",
        "month":    "01",
        "day":      "15",
        "ext":      "pdf",
        "category": "resume",
    }

    expander := template.NewExpander(placeholders)

    // 展开路径模板
    path, err := expander.ExpandPath("Documents/{category}/{year}/{month}")
    if err != nil {
        panic(err)
    }

    fmt.Println(path)
    // 输出: Documents/resume/2024/01
}
```

### API 文档

#### NewExpander

创建新的模板展开器。

```go
func NewExpander(placeholders map[string]string) *Expander
```

**参数**：

- `placeholders`：占位符映射表

**返回**：

- `*Expander`：模板展开器实例

#### ExpandPath

展开路径模板。

```go
func (e *Expander) ExpandPath(template string) (string, error)
```

**参数**：

- `template`：包含占位符的路径模板

**返回**：

- `string`：展开后的路径
- `error`：错误信息

**示例**：

```go
// 简单占位符
path, _ := expander.ExpandPath("Pictures/{year}/{month}")
// 结果: Pictures/2024/01

// 多个占位符
path, _ := expander.ExpandPath("Documents/{category}/{year}-{month}-{day}.{ext}")
// 结果: Documents/resume/2024-01-15.pdf

// 嵌套路径
path, _ := expander.ExpandPath("{category}/{year}/{month}/{day}")
// 结果: resume/2024/01/15
```

#### Expand

展开字符串模板（通用方法）。

```go
func (e *Expander) Expand(template string) (string, error)
```

**参数**：

- `template`：包含占位符的字符串模板

**返回**：

- `string`：展开后的字符串
- `error`：错误信息

### 错误处理

模板展开可能返回以下错误：

- **未知占位符**：模板中包含未定义的占位符
- **格式错误**：占位符格式不正确（如缺少闭合括号）

```go
path, err := expander.ExpandPath("Documents/{unknown}")
if err != nil {
    // 处理错误：未知占位符 "unknown"
}
```

### 扩展占位符

如果需要添加新的占位符：

1. 在调用 `NewExpander` 时添加到 placeholders 映射
2. 或者在创建后动态添加：

```go
expander := template.NewExpander(map[string]string{
    "year": "2024",
})

// 动态添加占位符
expander.Set("custom", "value")

path, _ := expander.ExpandPath("path/{custom}")
// 结果: path/value
```

### 测试

运行测试：

```bash
go test ./pkg/template -v
```

测试覆盖率：

```bash
go test ./pkg/template -cover
```

### 性能

模板展开是一个轻量级操作，性能开销很小：

- 时间复杂度：O(n)，n 为模板长度
- 空间复杂度：O(n)，n 为展开后字符串长度

### 最佳实践

1. **复用 Expander**：如果需要展开多个模板，复用同一个 Expander 实例
2. **验证占位符**：在展开前验证所有必需的占位符都已定义
3. **错误处理**：始终检查展开操作的错误返回值
4. **路径清理**：展开后使用 `filepath.Clean` 清理路径

### 使用场景

1. **文件整理规则**：根据文件元数据生成目标路径

   ```go
   target := "Documents/{category}/{year}/{month}"
   ```

2. **日志文件命名**：生成带日期的日志文件名

   ```go
   logFile := "logs/{year}-{month}-{day}.log"
   ```

3. **备份路径生成**：创建带时间戳的备份目录
   ```go
   backupDir := "backups/{year}/{month}/{day}"
   ```

### 限制

1. 占位符必须使用 `{name}` 格式
2. 占位符名称不能包含特殊字符
3. 不支持嵌套占位符（如 `{{name}}`）
4. 不支持条件展开或循环

### 未来计划

- [ ] 支持默认值：`{name:default}`
- [ ] 支持格式化：`{date:YYYY-MM-DD}`
- [ ] 支持条件展开：`{?category:path}`
- [ ] 支持函数调用：`{upper:name}`

## 添加新包

如果需要添加新的公共包：

1. 在 `pkg/` 下创建新目录
2. 实现功能并添加测试
3. 添加 README 或 godoc 注释
4. 更新此文档

## 设计原则

1. **通用性**：包应该是通用的，不依赖项目特定逻辑
2. **独立性**：包之间应该相互独立
3. **文档化**：提供清晰的 API 文档和使用示例
4. **测试覆盖**：确保高测试覆盖率
5. **向后兼容**：API 变更应保持向后兼容

## 参考资料

- [Go 包设计指南](https://golang.org/doc/effective_go.html#package-names)
- [项目架构文档](../docs/ARCHITECTURE.md)
