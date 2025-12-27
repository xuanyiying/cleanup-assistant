# Cleanup CLI v1.0.0

智能文件整理命令行工具，通过本地 Ollama 模型实现文件的智能分类、重命名和归档。

## 🎉 新功能

- 🤖 **AI 驱动的文件分析** - 集成 Ollama 本地模型
- 🏷️ **智能文件名识别** - 自动识别无意义文件名并重命名
- 📁 **灵活的规则引擎** - 支持自定义文件整理规则
- 🔄 **完整的事务管理** - 所有操作可撤销
- 📊 **详细的控制台输出** - 实时显示整理进度
- 🗑️ **安全删除** - 文件移至回收站而非永久删除

## 📦 安装

### macOS

#### 使用安装包（推荐）

下载 `cleanup-cli-1.0.0.pkg`，双击安装。

#### 使用 tar.gz

```bash
# 下载并解压
tar -xzf cleanup-cli-1.0.0-darwin.tar.gz
cd cleanup-cli-1.0.0

# 运行安装脚本
./install.sh
```

#### 使用 Homebrew

```bash
brew tap user/cleanup
brew install cleanup
```

### Linux

```bash
# 下载二进制文件
wget https://github.com/user/cleanup-cli/releases/download/v1.0.0/cleanup-linux-amd64

# 安装
sudo mv cleanup-linux-amd64 /usr/local/bin/cleanup
sudo chmod +x /usr/local/bin/cleanup
```

## 🚀 快速开始

1. **安装 Ollama**

   ```bash
   # 访问 https://ollama.ai 下载安装
   ollama serve
   ollama pull llama3.2
   ```

2. **运行 Cleanup**

   ```bash
   # 交互模式
   cleanup

   # 整理指定目录
   cleanup organize ~/Downloads
   ```

## 📋 系统要求

- macOS 10.15+ 或 Linux
- [Ollama](https://ollama.ai) 已安装
- 至少 8GB RAM
- 10GB 可用磁盘空间

## 📚 文档

- [完整文档](https://github.com/user/cleanup-cli/blob/main/README.md)
- [安装指南](https://github.com/user/cleanup-cli/blob/main/INSTALL.md)
- [配置示例](https://github.com/user/cleanup-cli/blob/main/.cleanuprc.yaml)

## 🐛 已知问题

无

## 🙏 致谢

感谢所有贡献者和测试者！

## 📄 许可证

MIT License

---

## 📦 下载

| 平台    | 架构      | 文件                                    | SHA256      |
| ------- | --------- | --------------------------------------- | ----------- |
| macOS   | Universal | [cleanup-cli-1.0.0-darwin.tar.gz](link) | `sha256sum` |
| macOS   | Installer | [cleanup-cli-1.0.0.pkg](link)           | `sha256sum` |
| Linux   | amd64     | [cleanup-linux-amd64](link)             | `sha256sum` |
| Windows | amd64     | [cleanup-windows-amd64.exe](link)       | `sha256sum` |

## 🔐 校验

```bash
# macOS
shasum -a 256 cleanup-cli-1.0.0-darwin.tar.gz

# Linux
sha256sum cleanup-linux-amd64
```
