# ccenv

[English](README_en.md)

管理 Claude Code 环境变量的 CLI 工具。通过命名 profile 快速切换供应商，切换只影响当前终端，其他 session 不受影响。

## 为什么需要 ccenv？

Claude Code 从 `~/.claude/settings.json` 读取配置。当你需要在不同供应商（官方 API、代理、自建中转）之间切换时，直接改 `settings.json` 会全局生效——所有正在运行的 Claude Code session 都会被影响，可能导致正在跑的任务中断。

**ccenv 让切换只影响你选择的终端。** 其他终端完全不受影响。

## 工作原理

```
~/.claude/settings.json         # Claude Code 配置（ccenv 从这里读取 env）
~/.claude/ccenv/<name>.profile  # 存储的 profile（可被 source 的 shell 脚本）
~/.claude/ccenv.activate        # 符号链接 → 当前激活的 profile
```

- `ccenv use <name>` 创建符号链接 `~/.claude/ccenv.activate` 指向选定的 profile
- 在 shell 中 `source` 这个符号链接即可加载环境变量
- `use` 不会修改 `settings.json`——只有 `save` 会从中读取
- 每个终端独立 `source` 自己需要的 profile

## 安装

### 方式一：下载预编译二进制（推荐）

前往 [GitHub Releases](https://github.com/ekeyme/claude-code-env/releases) 下载对应平台的压缩包。

```bash
# Linux amd64 示例
tar xzf ccenv_*_linux_amd64.tar.gz
mkdir -p ~/.local/bin
cp ccenv ~/.local/bin/
```

### 方式二：Go Install

```bash
go install github.com/ekeyme/claude-code-env/cmd/ccenv@latest
```

### 方式三：从源码构建

```bash
git clone https://github.com/ekeyme/claude-code-env.git
cd claude-code-env
make install
```

`make install` 会做以下事情：
1. 构建二进制 → `~/.local/bin/ccenv`
2. 安装辅助脚本：
   - `~/.local/bin/ccenv-claude` — 按指定 profile 启动 claude
   - `~/.local/bin/vscode-claude-wrapper.sh` — VSCode Claude Code 插件集成
   - `~/.claude/ccenv.deactivate` — 清除 ANTHROPIC_ 环境变量

> **注意：** 如果下载的是预编译二进制，辅助脚本在 release 压缩包中，需手动安装。

```bash
cp ccenv ccenv-claude vscode-claude-wrapper.sh ~/.local/bin/
chmod +x ~/.local/bin/ccenv ~/.local/bin/ccenv-claude ~/.local/bin/vscode-claude-wrapper.sh
mkdir -p ~/.claude
cp ccenv.deactivate ~/.claude/
```

### 安装后配置

确保 `~/.local/bin` 在 `PATH` 中：

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## 快速开始

```bash
# 1. 先在 Claude Code 中配置好供应商（编辑 ~/.claude/settings.json 的 env 字段）

# 2. 保存当前配置为 profile
ccenv save official

# 3. 切换到另一个供应商后，再保存一个
ccenv save proxy

# 4. 在当前终端激活某个 profile
ccenv use proxy
source ~/.claude/ccenv.activate

# 5. 这个终端使用 proxy，其他终端不受影响
```

## 命令

```
ccenv status              显示 settings.json env、已激活 profile、当前 shell ANTHROPIC_ 变量
ccenv save <name>         将 settings.json 当前 env 保存为 profile
ccenv use <name>          激活 profile（创建符号链接）
ccenv list                列出所有 profile（标注已激活）
ccenv show <name>         显示 profile 详情
ccenv delete <name>       删除 profile（如果是当前激活的则自动取消激活）
ccenv rename <old> <new>  重命名 profile（如果已激活则自动更新符号链接）
ccenv -v                  显示版本信息
```

### 命令示例

```bash
$ ccenv status
settings.json 官方 env:
  ANTHROPIC_AUTH_TOKEN=sk-a****
  ANTHROPIC_BASE_URL=https://api.anthropic.com

已激活 profile: proxy
  ANTHROPIC_AUTH_TOKEN=sk-p****
  ANTHROPIC_BASE_URL=https://proxy.example.com

当前环境中的 ANTHROPIC_ 变量:
  ANTHROPIC_AUTH_TOKEN=sk-p****
  ANTHROPIC_BASE_URL=https://proxy.example.com

$ ccenv save dev
已保存 profile 'dev' (2 个变量)

$ ccenv list
已保存的 profile:
  dev
  proxy (已激活)

$ ccenv show dev
profile 'dev':
  ANTHROPIC_AUTH_TOKEN=sk-d****
  ANTHROPIC_BASE_URL=https://dev.example.com

$ ccenv use dev
已激活 profile 'dev' (2 个变量)
生效方式: source /home/user/.claude/ccenv.activate

$ ccenv delete old-profile
已删除 profile 'old-profile'
```

## 使用场景

### 1. 多供应商切换

同时有官方 API 和区域代理，保存为不同 profile 自由切换：

```bash
ccenv save official    # 保存官方 API 凭据
ccenv save proxy-cn    # 保存代理凭据
ccenv use proxy-cn     # 在当前终端激活代理
source ~/.claude/ccenv.activate
```

### 2. 多终端同时使用不同供应商

终端 A 正在用官方 API 跑一个长任务。你打开终端 B 切换到另一个供应商——终端 A 完全不受影响：

```bash
# 终端 B
ccenv use proxy
source ~/.claude/ccenv.activate
claude
```

### 3. VSCode 集成

VSCode Claude Code 插件会以子进程方式启动 `claude`。正常情况下它继承系统环境——至于用哪个 profile，取决于你在终端里最后激活的是哪个。

**工作原理：** `vscode-claude-wrapper.sh` 替代了 `claude` 二进制的路径。每次 VSCode 启动 Claude 会话时，wrapper 会：

1. 清除已有的 `ANTHROPIC_` 变量（通过 `ccenv.deactivate`）
2. Source `~/.claude/ccenv.activate`（指向当前激活 profile 的符号链接）
3. `exec claude "$@"`——带着 profile 的环境变量启动真正的 `claude`

**切换 profile：** 只需在任意终端运行 `ccenv use <name>`，这会更新 `ccenv.activate` 符号链接。VSCode 中下一次启动的 Claude 会话就会自动使用新 profile。

配置方式：

```json
{
  "claude.coder.path": "/home/user/.local/bin/vscode-claude-wrapper.sh"
}
```

> **注意：** VSCode 中已在运行的 Claude 会话保持原有环境。新 profile 只对切换后启动的会话生效。

### 4. 使用 ccenv-claude 快速启动

`ccenv-claude` 一步完成 source profile + 启动 claude，不需要手动 `source`。

**工作原理：** 和 VSCode wrapper 相同的机制，但用于终端场景。先清除当前 `ANTHROPIC_` 变量，source 选定的 profile 文件，然后将其余参数传给 `claude`。

```bash
# 使用当前激活的 profile（读取 ccenv.activate 符号链接）
ccenv-claude

# 直接使用指定 profile（不受当前激活状态影响）
ccenv-claude proxy-cn

# 使用指定 profile 时向 Claude 传递参数
ccenv-claude proxy-cn --resume

# 交互式选择 profile
ccenv-claude list
```

适合想临时用某个供应商快速启动 Claude、但不想改变当前激活 profile 的场景。

### 5. 团队共享 Profile

Profile 文件就是普通的 shell 脚本，可以和团队共享：

```bash
# 导出
cp ~/.claude/ccenv/dev.profile ./shared/dev.profile

# 在其他机器上导入
cp ./shared/dev.profile ~/.claude/ccenv/dev.profile
```

## 安全

- **自动脱敏**：`status` 和 `show` 输出中，key 含 `token`/`key`/`secret`/`password`/`credential`/`private` 的变量值会自动脱敏（保留前 4 字符，其余替换为 `****`）
- **文件权限**：profile 文件权限 `0600`，目录权限 `0700`
- **原子写入**：先写临时文件再 rename，防止崩溃导致数据丢失

## 取消激活

清除当前 shell 中所有 `ANTHROPIC_` 环境变量：

```bash
source ~/.claude/ccenv.deactivate
```

## 许可证

[MIT](LICENSE)
