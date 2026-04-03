# ccenv - Claude Code Environment Manager

管理 `~/.claude/settings.json` 中 `env` 字段的 CLI 工具，支持命名 profile 的保存和切换。

## 项目结构

- `main.go` - 唯一源文件，包含所有逻辑
- `main_test.go` - 单元测试
- `go.mod` - Go module 定义，零外部依赖
- `Makefile` - 构建 / 安装 / 卸载

## 数据文件

- `~/.claude/settings.json` - 读写目标，仅操作 `env` 字段，其他字段原样保留
- `~/.claude/ccenv.config.json` - Profile 存储，格式 `{ "profileName": { "KEY": "VALUE" } }`

> **测试环境变量**: 可通过 `CCENV_CLAUDE_HOME` 环境变量指定自定义配置目录，用于测试时隔离环境。

## 环境变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `CCENV_CLAUDE_HOME` | 指定 Claude 配置目录（默认 `~/.claude`） | `CCENV_CLAUDE_HOME=/tmp/ccenv-test ./ccenv status` |

## 命令

- `ccenv status` - 显示当前 env
- `ccenv save <name>` - 保存当前 env 为 profile
- `ccenv use <name>` - 应用某个 profile
- `ccenv list` - 列出所有 profile
- `ccenv show <name>` - 显示 profile 详情
- `ccenv delete <name>` - 删除某个 profile
- `ccenv rename <old> <new>` - 重命名 profile

## 测试

```bash
# 运行单元测试
go test -v

# 运行性能测试
go test -bench=.
```

## 约束

- 纯 Go 标准库，零外部依赖
- 含 token/key/secret 的 key 自动脱敏（保留前4字符）
- 文件权限：settings.json 写入 0600，目录创建 0700