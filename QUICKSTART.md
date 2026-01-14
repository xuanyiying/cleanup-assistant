# Cleanup CLI - 快速开始

## 安装

### macOS 快速安装

```bash
# 1. 构建
make build

# 2. 安装
./install.sh

# 3. 验证
cleanup version
```

### 其他安装方式

#### 使用 Makefile

```bash
make install
```

#### 使用 Homebrew（本地）

```bash
make package-tar
brew install --formula ./Formula/cleanup.rb
```

#### 创建安装包

```bash
./scripts/package.sh
# 生成 dist/cleanup-cli-1.0.0.pkg
```

### 系统要求

- macOS 10.15+ / Linux
- [Ollama](https://ollama.ai) - AI 模型运行环境
- 推荐: 8GB+ RAM

### 安装 Ollama

```bash
# 访问 https://ollama.ai 下载
# 或使用 Homebrew
brew install ollama

# 启动服务
ollama serve

# 拉取模型
ollama pull llama3.2
```

## 五分钟上手

### 1. 启动 Ollama

```bash
ollama serve
ollama pull llama3.2
```

### 2. 运行 Cleanup

```bash
# 交互模式
cleanup

# 或直接整理
cleanup organize ~/Downloads
```

### 3. 查看结果

```bash
# 查看历史
cleanup history

# 撤销操作
cleanup undo
```

## 常用命令

| 命令                                | 别名      | 说明         |
| ----------------------------------- | --------- | ------------ |
| `cleanup`                           |           | 进入交互模式 |
| `cleanup scan [path]`               | `s`, `sc` | 扫描目录     |
| `cleanup organize [path]`           | `o`, `org`| 整理文件     |
| `cleanup organize --dry-run [path]` | `o`       | 预览模式     |
| `cleanup junk scan`                 | `j s`     | 扫描垃圾文件 |
| `cleanup junk clean`                | `j c`     | 清理垃圾文件 |
| `cleanup undo [txn-id]`             | `u`       | 撤销操作     |
| `cleanup history`                   | `h`, `hist`| 查看历史     |
| `cleanup version`                   | `v`       | 查看版本     |
| `cleanup --help`                    |           | 查看帮助     |

## 排除文件和文件夹

```bash
# 排除特定扩展名
cleanup scan --exclude-ext log,tmp

# 排除文件模式
cleanup organize --exclude-pattern "*.bak,*~"

# 排除目录
cleanup organize --exclude-dir .git,node_modules

# 组合使用
cleanup organize ~/Projects \
  --exclude-ext log,tmp \
  --exclude-pattern "*.bak" \
  --exclude-dir .git,node_modules,dist
```

## 配置文件

位置: `~/.cleanuprc.yaml`

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

## 使用示例

### 场景 1: 整理下载文件夹

```bash
cleanup organize ~/Downloads
```

结果：

- PDF → `Documents/PDF/`
- 图片 → `Pictures/YYYY/MM/`
- 视频 → `Videos/YYYY/`
- 无意义文件名 → AI 重命名后分类

### 场景 2: 预览模式

```bash
cleanup organize --dry-run ~/Downloads
```

查看将要执行的操作，不实际修改文件。

### 场景 3: 整理项目目录

```bash
cleanup organize ~/Projects \
  --exclude-dir .git,node_modules,dist \
  --exclude-ext log,tmp
```

自动跳过版本控制文件和构建产物。

### 场景 4: 整理文档

```bash
cleanup organize ~/Documents \
  --exclude-pattern "*.bak,*~"
```

整理文档并跳过备份文件。

### 场景 5: 系统清理

```bash
# 扫描系统垃圾（缓存、临时文件等）
cleanup junk scan

# 执行清理（默认移至回收站）
cleanup junk clean
```

### 场景 6: 撤销操作

```bash
# 查看历史
cleanup history

# 撤销最后一次操作
cleanup undo

# 撤销指定操作
cleanup undo txn_1234567890
```

## 工作流程

```
1. 扫描文件
   ↓
2. 评估文件名质量
   ↓
3. 如果文件名无意义 → AI 分析内容 → 生成新文件名
   ↓
4. 匹配规则
   ↓
5. 生成操作计划
   ↓
6. 执行整理（重命名 + 移动到分类文件夹）
   ↓
7. 记录事务（可撤销）
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
# 使用卸载脚本
./uninstall.sh

# 或使用 Makefile
make uninstall

# 或使用 Homebrew
brew uninstall cleanup
```

## 更多资源

- 📖 [完整文档](README.md)
- 💡 [示例脚本](examples/demo.sh)
- ⚙️ [配置示例](.cleanuprc.yaml)

---

**提示**: 首次使用建议先用 `--dry-run` 模式预览效果！
