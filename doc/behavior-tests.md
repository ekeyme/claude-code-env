# ccenv 行为测试场景

> 验证 ccenv 管理 `~/.claude/settings.json` 中 env 字段的各项行为。
> 目标文件：`~/.claude/settings.json`（env 字段）、`~/.claude/ccenv.config.json`（profile 存储）

---

## 测试前置数据说明

| 数据 | 说明 |
|------|------|
| settings.json | Claude Code 配置文件，ccenv 仅操作其中的 `env` 字段，其他字段（permissions、hooks 等）原样保留 |
| ccenv.config.json | ccenv 自己的 profile 存储，格式 `{ "profileName": { "KEY": "VALUE" } }` |

> 注：测试前需确保 `~/.claude/` 目录存在且 `settings.json` 为合法 JSON。

---

## Feature: 基础命令

### Scenario S1: 无参数运行显示帮助信息
```gherkin
When  运行 ccenv 不带参数
Then  退出码为 1
And   输出包含所有命令说明（status/save/use/list/show/delete/rename）
```

### Scenario S2: 未知命令显示错误
```gherkin
When  运行 "ccenv unknown"
Then  退出码为 1
And   stderr 包含 "未知命令: unknown"
```

---

## Feature: ccenv status — 显示当前 env

### Scenario S3: 显示当前 env 内容（含脱敏）
```gherkin
Given settings.json 的 env 为:
      {
        "ANTHROPIC_AUTH_TOKEN": "sk-ant-1234567890abcdef",
        "BASE_URL": "https://api.example.com"
      }
When  运行 "ccenv status"
Then  退出码为 0
And   输出包含 "当前 env:"
And   输出包含 "ANTHROPIC_AUTH_TOKEN=sk-a****"
And   输出包含 "BASE_URL=https://api.example.com"
```

### Scenario S4: env 为空时显示空提示
```gherkin
Given settings.json 的 env 为 {}
When  运行 "ccenv status"
Then  退出码为 0
And   输出包含 "(空)"
```

### Scenario S5: settings.json 不存在时视为空 env
```gherkin
Given settings.json 不存在
When  运行 "ccenv status"
Then  退出码为 0
And   输出包含 "(空)"
```

### Scenario S6: settings.json 为空白文件时不报错
```gherkin
Given settings.json 文件内容为空（0 字节）
When  运行 "ccenv status"
Then  退出码为 0
And   输出包含 "(空)"
```

---

## Feature: 脱敏规则

### Scenario S7: 含 token/key/secret/password/credential/private 的值自动脱敏
```gherkin
Given settings.json 的 env 为:
      {
        "MY_TOKEN": "abcdefghijklmn",
        "API_KEY": "1234567890",
        "APP_SECRET": "secret_value_here",
        "USER_PASSWORD": "mypassword123",
        "AWS_CREDENTIAL": "cred_data_here",
        "SSH_PRIVATE_KEY": "private_data_here"
      }
When  运行 "ccenv status"
Then  输出包含 "MY_TOKEN=abcd****"
And   输出包含 "API_KEY=1234****"
And   输出包含 "APP_SECRET=secr****"
And   输出包含 "USER_PASSWORD=mypa****"
And   输出包含 "AWS_CREDENTIAL=cred****"
And   输出包含 "SSH_PRIVATE_KEY=priv****"
```

### Scenario S8: 敏感 key 的值长度 ≤ 4 时完全遮蔽
```gherkin
Given settings.json 的 env 为:
      {
        "MY_TOKEN": "abc",
        "API_KEY": "12",
        "APP_SECRET": ""
      }
When  运行 "ccenv status"
Then  输出包含 "MY_TOKEN=****"
And   输出包含 "API_KEY=****"
And   输出包含 "APP_SECRET=****"
```

### Scenario S9: 非敏感 key 不脱敏
```gherkin
Given settings.json 的 env 为:
      {
        "APP_NAME": "my-app",
        "BASE_URL": "https://example.com",
        "DEBUG": "true"
      }
When  运行 "ccenv status"
Then  输出包含 "APP_NAME=my-app"
And   输出包含 "BASE_URL=https://example.com"
And   输出包含 "DEBUG=true"
```

### Scenario S10: 多字节字符值正确脱敏（不截断 Unicode）
```gherkin
Given settings.json 的 env 为:
      {
        "API_KEY": "密码内容在这里"
      }
When  运行 "ccenv status"
Then  输出包含 "API_KEY=密码****"
```

### Scenario S11: 输出按 key 字母排序
```gherkin
Given settings.json 的 env 为:
      {
        "Z_VAR": "z",
        "A_VAR": "a",
        "M_VAR": "m"
      }
When  运行 "ccenv status"
Then  输出中 A_VAR 出现在 M_VAR 之前
And   输出中 M_VAR 出现在 Z_VAR 之前
```

---

## Feature: ccenv save — 保存 profile

### Scenario S12: 保存当前 env 为 profile
```gherkin
Given settings.json 的 env 为:
      {
        "FOO": "bar",
        "BAZ": "qux"
      }
When  运行 "ccenv save my-profile"
Then  退出码为 0
And   输出包含 "已保存 profile 'my-profile' (2 个变量)"
And   ccenv.config.json 中 profile "my-profile" 包含 FOO=bar 和 BAZ=qux
```

### Scenario S13: 保存空 env 的 profile
```gherkin
Given settings.json 的 env 为 {}
When  运行 "ccenv save empty-profile"
Then  退出码为 0
And   输出包含 "已保存 profile 'empty-profile' (0 个变量)"
```

### Scenario S14: 保存同名 profile 覆盖已有内容
```gherkin
Given settings.json 的 env 为:
      {
        "NEW": "value1"
      }
And   ccenv.config.json 中已存在 profile "dup" 包含 {"OLD": "value2"}
When  运行 "ccenv save dup"
Then  退出码为 0
And   ccenv.config.json 中 profile "dup" 仅包含 NEW=value1（OLD 已被覆盖）
```

### Scenario S15: save 不指定名称报错
```gherkin
When  运行 "ccenv save"
Then  退出码为 1
And   stderr 包含 "请指定 profile 名称"
```

---

## Feature: ccenv use — 应用 profile

### Scenario S16: 应用 profile 覆盖当前 env
```gherkin
Given ccenv.config.json 中已存在 profile "staging" 包含:
      {
        "API": "https://staging.io"
      }
And   settings.json 的 env 为:
      {
        "API": "https://prod.io"
      }
When  运行 "ccenv use staging"
Then  退出码为 0
And   输出包含 "已切换到 profile 'staging' (1 个变量)"
And   settings.json 的 env 变为 {"API": "https://staging.io"}
```

### Scenario S17: use 不存在的 profile 报错
```gherkin
Given ccenv.config.json 中不存在 profile "nonexistent"
When  运行 "ccenv use nonexistent"
Then  退出码为 1
And   stderr 包含 "profile 'nonexistent' 不存在"
```

### Scenario S18: use 不指定名称报错
```gherkin
When  运行 "ccenv use"
Then  退出码为 1
And   stderr 包含 "请指定 profile 名称"
```

### Scenario S19: use 后 settings.json 其他字段保留不变
```gherkin
Given ccenv.config.json 中已存在 profile "myenv" 包含 {"FOO": "bar"}
And   settings.json 包含 permissions、hooks 等非 env 字段
When  运行 "ccenv use myenv"
Then  settings.json 中 permissions、hooks 等字段原样保留
And   仅 env 字段被替换
```

### Scenario S20: 应用空 profile 清空当前 env
```gherkin
Given ccenv.config.json 中已存在 profile "empty" 包含 {}
And   settings.json 的 env 为 {"FOO": "bar"}
When  运行 "ccenv use empty"
Then  settings.json 的 env 变为 {}
```

---

## Feature: ccenv list — 列出 profile

### Scenario S21: 列出所有已保存的 profile
```gherkin
Given ccenv.config.json 中已存在 profile "alpha" 包含 2 个变量
And   ccenv.config.json 中已存在 profile "beta" 包含 3 个变量
When  运行 "ccenv list"
Then  退出码为 0
And   输出包含 "alpha (2 个变量)"
And   输出包含 "beta (3 个变量)"
```

### Scenario S22: 没有 profile 时显示空提示
```gherkin
Given ccenv.config.json 不存在
When  运行 "ccenv list"
Then  退出码为 0
And   输出包含 "没有已保存的 profile"
```

### Scenario S23: 列表按名称字母排序
```gherkin
Given ccenv.config.json 中已存在 profile "z-profile"、"a-profile"、"m-profile"
When  运行 "ccenv list"
Then  输出中 a-profile 出现在 m-profile 之前
And   输出中 m-profile 出现在 z-profile 之前
```

---

## Feature: ccenv show — 显示 profile 详情

### Scenario S24: 显示 profile 的详细内容（含脱敏）
```gherkin
Given ccenv.config.json 中已存在 profile "prod" 包含:
      {
        "API_ENDPOINT": "https://prod.io",
        "DB_SECRET": "supersecret123"
      }
When  运行 "ccenv show prod"
Then  退出码为 0
And   输出包含 "profile 'prod':"
And   输出包含 "API_ENDPOINT=https://prod.io"
And   输出包含 "DB_SECRET=supe****"
```

### Scenario S25: show 不存在的 profile 报错
```gherkin
When  运行 "ccenv show nonexistent"
Then  退出码为 1
And   stderr 包含 "profile 'nonexistent' 不存在"
```

### Scenario S26: show 不指定名称报错
```gherkin
When  运行 "ccenv show"
Then  退出码为 1
And   stderr 包含 "请指定 profile 名称"
```

---

## Feature: ccenv delete — 删除 profile

### Scenario S27: 删除已存在的 profile
```gherkin
Given ccenv.config.json 中已存在 profile "to-delete"
When  运行 "ccenv delete to-delete"
Then  退出码为 0
And   输出包含 "已删除 profile 'to-delete'"
And   ccenv.config.json 中 profile "to-delete" 不再存在
```

### Scenario S28: delete 不存在的 profile 报错
```gherkin
Given ccenv.config.json 中不存在 profile "ghost"
When  运行 "ccenv delete ghost"
Then  退出码为 1
And   stderr 包含 "profile 'ghost' 不存在"
```

### Scenario S29: delete 不指定名称报错
```gherkin
When  运行 "ccenv delete"
Then  退出码为 1
And   stderr 包含 "请指定 profile 名称"
```

---

## Feature: ccenv rename — 重命名 profile

### Scenario S30: 重命名 profile
```gherkin
Given ccenv.config.json 中已存在 profile "old-name" 包含 {"FOO": "bar"}
When  运行 "ccenv rename old-name new-name"
Then  退出码为 0
And   输出包含 "已将 profile 'old-name' 重命名为 'new-name'"
And   ccenv.config.json 中 profile "old-name" 不再存在
And   ccenv.config.json 中 profile "new-name" 包含 FOO=bar
```

### Scenario S31: rename 目标名称已存在时报错
```gherkin
Given ccenv.config.json 中已存在 profile "source" 和 "target"
When  运行 "ccenv rename source target"
Then  退出码为 1
And   stderr 包含 "profile 'target' 已存在"
```

### Scenario S32: rename 源 profile 不存在报错
```gherkin
Given ccenv.config.json 中不存在 profile "nonexistent"
When  运行 "ccenv rename nonexistent other"
Then  退出码为 1
And   stderr 包含 "profile 'nonexistent' 不存在"
```

### Scenario S33: rename 缺少参数报错
```gherkin
When  运行 "ccenv rename"
Then  退出码为 1
And   stderr 包含 "请指定旧名称和新名称"
```

```gherkin
When  运行 "ccenv rename only-old"
Then  退出码为 1
And   stderr 包含 "请指定旧名称和新名称"
```

---

## Feature: 完整工作流（端到端）

### Scenario S34: 典型工作流 — 保存 → 列出 → 切换 → 验证 → 删除
```gherkin
Given settings.json 的 env 为:
      {
        "ANTHROPIC_AUTH_TOKEN": "sk-ant-abcdef123",
        "BASE_URL": "https://dev.api.io"
      }
When  运行 "ccenv save dev"
Then  profile "dev" 已保存

When  运行 "ccenv list"
Then  输出包含 "dev (2 个变量)"

When  修改 settings.json 的 env 为:
      {
        "ANTHROPIC_AUTH_TOKEN": "sk-ant-xyz789",
        "BASE_URL": "https://prod.api.io"
      }
And   运行 "ccenv save prod"

When  运行 "ccenv use dev"
Then  settings.json 的 env 变为:
      {
        "ANTHROPIC_AUTH_TOKEN": "sk-ant-abcdef123",
        "BASE_URL": "https://dev.api.io"
      }

When  运行 "ccenv status"
Then  输出包含 "BASE_URL=https://dev.api.io"
And   输出包含 "ANTHROPIC_AUTH_TOKEN=sk-a****"

When  运行 "ccenv delete dev"
Then  profile "dev" 不再存在
And   profile "prod" 仍存在
```

---

## Feature: 数据安全与容错

### Scenario S35: 写入文件权限为 0600
```gherkin
When  运行 "ccenv save perm-test"
Then  settings.json 的文件权限为 0600
And   ccenv.config.json 的文件权限为 0600
```

### Scenario S36: ccenv.config.json 为空白文件时不报错
```gherkin
Given ccenv.config.json 文件内容为空（0 字节）
When  运行 "ccenv list"
Then  退出码为 0
And   输出包含 "没有已保存的 profile"
```

### Scenario S37: settings.json 中 env 含非字符串值时打印警告
```gherkin
Given settings.json 的 env 为:
      {
        "PORT": 8080,
        "API_KEY": "valid-string"
      }
When  运行 "ccenv status"
Then  stderr 包含 "警告" 和 "PORT"
And   输出包含 "API_KEY=****"（PORT 被跳过不显示）
```

---

## 关键验证点

| 验证点 | 对应场景 | 预期行为 |
|--------|----------|----------|
| 脱敏 - 6 种敏感关键词 | S7 | token/key/secret/password/credential/private 均脱敏，保留前 4 字符 |
| 脱敏 - 短值遮蔽 | S8 | 值长度 ≤ 4 时完全遮蔽为 **** |
| 脱敏 - 非敏感不处理 | S9 | 普通变量原样显示 |
| 脱敏 - Unicode 安全 | S10 | 多字节字符按 rune 截断，不破坏 UTF-8 |
| 输出排序 | S11 | 按 key 字母升序 |
| save 覆盖 | S14 | 同名 profile 直接覆盖，不报错 |
| use 互斥替换 | S16 | use 时整体替换 env，非合并 |
| use 其他字段保留 | S19 | settings.json 非 env 字段（permissions/hooks 等）原样保留 |
| 空白文件容错 | S6, S36 | settings.json 或 config.json 为空文件时不报错，视为空数据 |
| 非字符串值警告 | S37 | env 中非字符串值跳过并在 stderr 打印警告 |
| 文件权限 | S35 | 写入文件权限 0600 |
| 原子写入 | — | 通过 safeWriteFile 写入，临时文件放系统 temp 目录 |
