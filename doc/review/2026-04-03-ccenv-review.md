# ccenv 代码审查报告

日期：2026-04-03
审查范围：`main.go`、`Makefile`
审查维度：安全性、健壮性、代码质量

---

## 高优先级问题

### CRITICAL-1: maskValue 脱敏逻辑不充分

**文件**: `main.go:19-30`

**问题**:
- 值长度 ≤ 8 时完全不脱敏，短 token/key 直接暴露
- `len(value)` 按 UTF-8 字节计算，多字节字符（如中文）可能在中间截断产生无效 UTF-8
- 敏感关键词仅覆盖 `token`/`key`/`secret`，未覆盖 `password`/`credential`/`private`

```go
// 当前实现
if len(value) <= 8 {
    return value  // 短密钥直接暴露
}
return value[:8] + "****"  // 多字节字符可能截断
```

**修复方案**:
```go
func maskValue(key, value string) string {
    lower := strings.ToLower(key)
    if strings.Contains(lower, "token") ||
        strings.Contains(lower, "key") ||
        strings.Contains(lower, "secret") ||
        strings.Contains(lower, "password") ||
        strings.Contains(lower, "credential") ||
        strings.Contains(lower, "private") {
        runes := []rune(value)
        if len(runes) <= 4 {
            return "****"
        }
        return string(runes[:4]) + "****"
    }
    return value
}
```

---

### CRITICAL-2: 符号链接攻击

**文件**: `main.go:59-151`（readSettings, writeSettings, readProfiles, writeProfiles）

**问题**: 所有读写操作使用 `os.ReadFile` / `os.WriteFile`，会自动跟随符号链接。如果攻击者在 `~/.claude/` 下创建指向 `/etc/shadow` 等系统文件的 symlink：
- `readSettings` 可读取并输出敏感系统文件
- `writeSettings` 可覆写系统文件

**修复方案**: 读写前用 `os.Lstat` 检查是否为符号链接，发现时拒绝操作。

---

### CRITICAL-3: 非字符串 env 值静默丢弃

**文件**: `main.go:97-109`

**问题**: `getEnvFromSettings` 中，env 下的非字符串值（数字、布尔值、null）被静默忽略。用户执行 `save` → `use` 后这些值永久丢失，无任何警告。

```go
if s, ok := v.(string); ok {
    env[k] = s
}
// 非字符串值被静默丢弃
```

**修复方案**: 丢弃时输出 stderr 警告。

---

### CRITICAL-4: 非原子写入，崩溃可丢数据

**文件**: `main.go:77-94, 135-152`

**问题**: `os.WriteFile` 直接覆盖目标文件。写入过程中如果程序被中断（SIGKILL、断电、磁盘满），文件可能被截断为部分内容或空文件，导致 settings 或 profiles 数据丢失。

**修复方案**: 使用 write-to-temp-then-rename 模式，`os.Rename` 在同一文件系统上是原子的。

---

### CRITICAL-5: 空白文件解析直接报错

**文件**: `main.go:70, 128`

**问题**: `settings.json` 或 `ccenv.config.json` 存在但内容为空（0 字节或空白字符）时，`json.Unmarshal` 返回错误，所有命令无法使用。用户需要手动删除文件才能恢复。

**修复方案**: 解析前 `bytes.TrimSpace`，空内容回退到空 map。

---

### CRITICAL-6: profile 名称无校验

**文件**: `main.go:185`

**问题**: profile 名称仅检查空字符串，可注入 `.`、`..`、空格、换行、引号等特殊字符。虽然存储在 JSON 中不会路径穿越，但会导致输出格式混乱、手动编辑困难。

**修复方案**: 校验名称只允许 `[a-zA-Z0-9_-]`，长度限制 64 字符。注意这会引入 `regexp` 包依赖，也可手动遍历字符实现。

---

## 改进建议

### IMPROVE-1: read/write 函数严重重复（DRY）

**文件**: `main.go:54-94, 112-152`

**问题**: `readSettings`/`readProfiles` 和 `writeSettings`/`writeProfiles` 逻辑几乎完全相同，约 60 行重复代码。

**修复方案**: 抽取通用 `readJSONFile(filename, dest)` 和 `writeJSONFile(filename, data)` 函数。

---

### IMPROVE-2: readSettings 返回未使用的 []byte

**文件**: `main.go:54`

**问题**: 函数签名返回 `(map[string]interface{}, []byte, error)`，第二个返回值在所有调用方中均未被使用（全部用 `_` 接收）。

**修复方案**: 移除 `[]byte` 返回值，简化为 `(map[string]interface{}, error)`。

---

### IMPROVE-3: Makefile install 残留构建产物

**文件**: `Makefile:9-14`

**问题**: `install` 依赖 `build`，会在当前目录生成 `ccenv` 二进制文件，安装后残留。

**修复方案**: 直接 `go build -o $(INSTALL_DIR)/$(BINARY)` 到目标路径。

---

## 审查结论

代码整体结构清晰，错误处理覆盖面不错，使用了 `filepath.Join` 防止路径遍历。主要风险集中在：

1. **安全**：脱敏逻辑边界情况 + 符号链接攻击
2. **数据安全**：非原子写入 + 非字符串值静默丢失
3. **代码质量**：读写函数严重重复

建议优先修复 CRITICAL-1 ~ CRITICAL-4，IMPROVE 建议在重构时一并处理。

---

## 处理决定

| 编号 | 决定 | 说明 |
|------|------|------|
| CRITICAL-1 | **已修复** | 改用 rune 处理 Unicode，短值完全遮蔽为 `****`，增加 password/credential/private 关键词 |
| CRITICAL-2 | 不修复 | 记录风险。~/.claude 目录由用户自己控制，实际攻击场景有限 |
| CRITICAL-3 | **已修复** | 加 stderr 警告，不修复逻辑 |
| CRITICAL-4 | **已修复** | 实现 `safeWriteFile`，临时文件放系统 temp 目录（非 ~/.claude），验证后写入 |
| CRITICAL-5 | **已修复** | readSettings/readProfiles 解析前 bytes.TrimSpace，空内容回退空 map |
| CRITICAL-6 | 不修复 | 记录风险。JSON 存储不影响路径安全 |
| IMPROVE-1 | 忽略 | — |
| IMPROVE-2 | 忽略 | — |
| IMPROVE-3 | 忽略 | — |
