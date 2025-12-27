# Cleanup CLI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey)](https://github.com/user/cleanup-cli)

智能文件整理命令行工具，通过本地 Ollama 模型实现文件的智能分类、重命名和归档。

## 功能特性

- 🤖 **AI 驱动** - 集成本地 Ollama 模型，智能分析文件内容
- 🏷️ **智能重命名** - 自动识别无意义文件名，基于内容生成有意义的名称
- 📁 **自动分类** - 根据规则自动创建文件夹并移动文件到正确位置
- 📋 **规则引擎** - 支持自定义规则，灵活配置文件整理策略
- ↩️ **事务回滚** - 所有操作可撤销，安全可靠
- 🗑️ **安全删除** - 删除操作移至回收站，防止误删
- ⚡ **批量处理** - 支持并发处理大量文件
- 🚫 **智能排除** - 自动跳过版本控制、依赖包等不需要整理的文件
- 🖥️ **交互式界面** - 支持自然语言命令

## 工作流程

```
扫描文件 → 评估文件名 → AI 重命名（如需要）→ 分析文档场景 → 匹配规则 → 创建分类文件夹 → 移动文件 → 记录事务
```

**示例**：

```
untitled.txt (内容: "Meeting notes...")
  ↓ AI 分析
meeting-notes-jan-15.txt
  ↓ 场景分析
Category: meeting
  ↓ 匹配规则
Documents/meeting/meeting-notes-jan-15.txt
```

### 智能文件名识别

工具会自动评估文件名质量：

- **有意义的文件名** (如 `project-report-2024.pdf`) - 直接分类到对应文件夹
- **无意义的文件名** (如 `IMG_1234.jpg`, `untitled.txt`, `新建文档.docx`) - AI 分析内容生成新名称后分类
- **通用文件名** (如 `doc.txt`, `data.csv`) - 根据配置决定是否重命名

支持识别的无意义文件名模式：

- 时间戳命名：`IMG_1234.jpg`, `Screenshot_20240101.png`
- 通用名称：`untitled`, `新建文档`, `download`, `temp`
- 纯数字：`123456.pdf`, `20240101.txt`
- 单字符：`a.txt`, `1.doc`

### 文档场景分类

工具能够分析文档内容，识别其场景类型，进行智能分类：

**支持的场景类型**：

- 📄 **简历** (resume) - 个人简历、CV、求职文档
- 🎯 **面试** (interview) - 面试题、面试准备、面试笔记
- 📋 **会议** (meeting) - 会议记录、会议纪要、讨论记录
- 📊 **报告** (report) - 分析报告、工作报告、数据报告
- 💡 **提案** (proposal) - 项目提案、建议书、方案文档
- 📜 **合同** (contract) - 合同、协议、条款文档
- 💰 **发票** (invoice) - 发票、账单、收据
- 📖 **指南** (guide) - 指南、教程、说明书
- 📝 **笔记** (notes) - 笔记、备忘录、草稿

**工作原理**：

1. 扫描文档文件
2. 提取文档内容（最多 1500 字符）
3. 使用 AI 分析文档的主要用途和场景
4. 自动分类到对应的场景文件夹
5. 支持自定义分类规则

### 自动分类和文件夹管理

- ✅ 根据文件类型自动创建分类文件夹（如 `Documents/PDF/`, `Pictures/2024/01/`）
- ✅ 根据文档场景自动创建场景文件夹（如 `Documents/resume/`, `Documents/interview/`）
- ✅ 支持多级目录结构
- ✅ 支持日期模板（`{year}`, `{month}`, `{day}`）
- ✅ 支持场景模板（`{category}`）
- ✅ 自动处理文件名冲突

### 排除功能

默认排除常见的系统文件和开发文件：

- 系统文件：`.DS_Store`, `Thumbs.db`, `desktop.ini`
- 版本控制：`.git`, `.svn`, `.hg`
- 依赖包：`node_modules`, `__pycache__`, `vendor`

可通过命令行参数或配置文件自定义排除规则。

## 快速开始

```bash
# 1. 构建并安装
make build && ./install.sh

# 2. 启动 Ollama
ollama serve
ollama pull llama3.2

# 3. 整理文件
cleanup organize ~/Downloads
```

详细安装和使用说明请查看 [快速开始指南](QUICKSTART.md)。

## 使用方法

### 基本命令

```bash
# 扫描目录
cleanup scan ~/Downloads

# 整理文件
cleanup organize ~/Downloads

# 预览模式（不实际修改文件）
cleanup organize --dry-run ~/Downloads

# 撤销操作
cleanup undo

# 查看历史
cleanup history
```

### 排除文件和文件夹

```bash
# 排除特定扩展名
cleanup organize ~/Documents --exclude-ext log,tmp

# 排除文件模式
cleanup organize ~/Documents --exclude-pattern "*.bak,*~"

# 排除目录
cleanup organize ~/Projects --exclude-dir .git,node_modules,dist

# 组合使用
cleanup organize ~/Projects \
  --exclude-ext log,tmp \
  --exclude-pattern "*.bak" \
  --exclude-dir .git,node_modules
```

## 配置

配置文件位于 `~/.cleanuprc.yaml`：

```yaml
ollama:
  baseUrl: http://localhost:11434
  model: llama3.2
  timeout: 30s

rules:
  # 图片按日期整理
  - name: images-by-date
    priority: 100
    condition:
      type: extension
      value: jpg,jpeg,png,gif
      operator: match
    action:
      type: move
      target: "Pictures/{year}/{month}"

  # 文档按场景分类（简历、面试、会议等）
  - name: documents-by-scenario
    priority: 75
    condition:
      type: extension
      value: pdf,doc,docx,txt,md
      operator: match
    action:
      type: move
      target: "Documents/{category}"

  # PDF 文档
  - name: pdf-documents
    priority: 70
    condition:
      type: extension
      value: pdf
      operator: match
    action:
      type: move
      target: "Documents/PDF"

  # Office 文档
  - name: office-documents
    priority: 60
    condition:
      type: extension
      value: doc,docx,xls,xlsx,ppt,pptx
      operator: match
    action:
      type: move
      target: "Documents/Office"

  # 文本文件
  - name: text-files
    priority: 50
    condition:
      type: extension
      value: txt,md,rtf
      operator: match
    action:
      type: move
      target: "Documents/Text"

defaultStrategy:
  useAI: true
  createFolders: true
  conflictStrategy: suffix

# 排除配置
exclude:
  extensions:
    - log
    - tmp
    - cache
  patterns:
    - "*.bak"
    - "*.swp"
    - ".DS_Store"
  dirs:
    - .git
    - .svn
    - node_modules
    - __pycache__
```

### 规则配置

#### 条件类型

| 类型        | 说明       | 示例                  |
| ----------- | ---------- | --------------------- |
| `extension` | 文件扩展名 | `jpg,png,gif`         |
| `pattern`   | 文件名模式 | `*.log` (glob) 或正则 |
| `size`      | 文件大小   | `1MB`, `100KB`        |
| `date`      | 修改日期   | `2024-01-01`          |

#### 操作符

- `match`, `eq` - 匹配
- `ne` - 不匹配
- `gt`, `lt`, `gte`, `lte` - 大小比较
- `before`, `after` - 日期比较

#### 模板占位符

| 占位符       | 说明         | 示例                                                                          |
| ------------ | ------------ | ----------------------------------------------------------------------------- |
| `{year}`     | 年份 (4 位)  | 2024                                                                          |
| `{month}`    | 月份 (2 位)  | 01                                                                            |
| `{day}`      | 日期 (2 位)  | 15                                                                            |
| `{ext}`      | 文件扩展名   | pdf                                                                           |
| `{category}` | 文档场景分类 | resume, interview, meeting, report, proposal, contract, invoice, guide, notes |

**场景分类说明**：

- `resume` - 简历、CV、求职文档
- `interview` - 面试题、面试准备、面试笔记
- `meeting` - 会议记录、会议纪要、讨论记录
- `report` - 分析报告、工作报告、数据报告
- `proposal` - 项目提案、建议书、方案文档
- `contract` - 合同、协议、条款文档
- `invoice` - 发票、账单、收据
- `guide` - 指南、教程、说明书
- `notes` - 笔记、备忘录、草稿

### 冲突处理策略

- `skip` - 跳过冲突文件
- `suffix` - 添加数字后缀
- `overwrite` - 覆盖已有文件
- `prompt` - 交互式询问

## 示例场景

### 整理下载文件夹

```bash
cleanup organize ~/Downloads
```

结果：

- PDF → `Documents/PDF/`
- 图片 → `Pictures/2024/01/`
- 视频 → `Videos/2024/`
- 无意义文件名 → AI 重命名后分类

### 整理项目目录

```bash
cleanup organize ~/Projects \
  --exclude-dir .git,node_modules,dist \
  --exclude-ext log,tmp
```

自动跳过版本控制文件和构建产物。

### 整理文档

```bash
cleanup organize ~/Documents \
  --exclude-pattern "*.bak,*~"
```

整理文档并跳过备份文件。

## 开发

```bash
# 运行测试
go test ./...

# 运行特定测试
go test ./internal/analyzer -v

# 运行演示脚本
./examples/demo.sh

# 测试整理功能
./examples/test-organize.sh
```

## 项目结构

```
cleanup-cli/
├── cmd/cleanup/          # CLI 入口
├── internal/
│   ├── analyzer/         # 文件分析器
│   ├── config/           # 配置管理
│   ├── ollama/           # Ollama 客户端
│   ├── organizer/        # 文件整理器
│   ├── rules/            # 规则引擎
│   ├── shell/            # 交互式界面
│   └── transaction/      # 事务管理
├── pkg/template/         # 模板引擎
├── examples/             # 示例脚本
└── integration_test/     # 集成测试
```

## 故障排除

### Ollama 连接失败

```bash
# 检查服务
ps aux | grep ollama

# 启动服务
ollama serve

# 测试连接
curl http://localhost:11434/api/tags
```

### 权限问题

```bash
# 使用 sudo 安装
sudo make install

# 或手动设置权限
sudo chmod +x /usr/local/bin/cleanup
```

### 找不到命令

```bash
# 检查 PATH
echo $PATH

# 添加到 PATH
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## 卸载

```bash
./uninstall.sh
```

## 更多资源

- 📖 [快速开始](QUICKSTART.md)
- 💡 [示例脚本](examples/)
- ⚙️ [配置示例](.cleanuprc.yaml)
- 📝 [更新日志](CHANGELOG.md)

## License

MIT
