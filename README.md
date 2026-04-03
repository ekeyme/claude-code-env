# ccenv

管理 Claude Code 环境变量的 CLI 工具。通过命名 profile 快速切换 `~/.claude/settings.json` 中的 `env` 配置。

## 安装

```bash
# 构建
make build

# 安装到 ~/.local/bin
make install

# 确保 ~/.local/bin 在 PATH 中
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

卸载：

```bash
make uninstall
```

## 命令

```
ccenv status              显示当前 settings.json 中的 env
ccenv save <name>         将当前 env 保存为命名 profile
ccenv use <name>          应用某个 profile，覆盖当前 env
ccenv list                列出所有已保存的 profile
ccenv show <name>         显示某个 profile 的详细内容
ccenv delete <name>       删除某个 profile
ccenv rename <old> <new>  重命名某个 profile
```

## 使用示例

```bash
# 查看当前环境变量
$ ccenv status
当前 env:
  ANTHROPIC_AUTH_TOKEN=sk-a****
  ANTHROPIC_BASE_URL=https://api.example.com

# 保存当前配置为 dev profile
$ ccenv save dev
已保存 profile 'dev' (2 个变量)

# 切换到另一套配置后保存为 prod
$ ccenv save prod
已保存 profile 'prod' (2 个变量)

# 列出所有 profile
$ ccenv list
已保存的 profile:
  dev (2 个变量)
  prod (2 个变量)

# 切换回 dev 环境
$ ccenv use dev
已切换到 profile 'dev' (2 个变量)
```

## 数据存储

| 文件 | 说明 |
|------|------|
| `~/.claude/settings.json` | Claude Code 配置，ccenv 仅操作 `env` 字段，其他字段原样保留 |
| `~/.claude/ccenv.config.json` | ccenv 的 profile 存储 |

## 安全

- 含 `token`/`key`/`secret`/`password`/`credential`/`private` 的变量值自动脱敏（`status` 和 `show` 命令），保留前 4 个字符，其余替换为 `****`
- 写入文件权限 `0600`，目录权限 `0700`
- 写入时先写系统临时文件验证后再写入目标，防止崩溃导致数据丢失
