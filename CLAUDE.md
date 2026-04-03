# ccenv - Claude Code Environment Manager

管理 `~/.claude/settings.json` 中 `env` 字段的 CLI 工具，支持命名 profile 的保存和切换。

## 项目结构

- `main.go` - 唯一源文件，包含所有逻辑
- `go.mod` - Go module 定义，零外部依赖
- `Makefile` - 构建 / 安装 / 卸载

## 数据文件

- `~/.claude/settings.json` - 读写目标，仅操作 `env` 字段，其他字段原样保留
- `~/.claude/ccenv.config.json` - Profile 存储，格式 `{ "profileName": { "KEY": "VALUE" } }`

## 命令

- `ccenv status` - 显示当前 env
- `ccenv save <name>` - 保存当前 env 为 profile
- `ccenv use <name>` - 应用 profile
- `ccenv list` - 列出所有 profile
- `ccenv show <name>` - 显示 profile 详情
- `ccenv delete <name>` - 删除 profile
- `ccenv rename <old> <new>` - 重命名 profile

## 约束

- 纯 Go 标准库，零外部依赖
- 含 token/key/secret 的 key 自动脱敏（保留前8位）
- 文件权限：settings.json 写入 0600，目录创建 0700
