package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	appName      = "ccenv"
	settingsFile = "settings.json"
	activateFile = "ccenv.activate"
	homeEnvVar   = "CCENV_CLAUDE_HOME"
)

// 通过 -ldflags 在构建时注入
var (
	version   = "dev"
	buildTime = "unknown"
)

var sensitiveWords = []string{"token", "key", "secret", "password", "credential", "private"}

func maskValue(key, value string) string {
	lower := strings.ToLower(key)
	for _, w := range sensitiveWords {
		if strings.Contains(lower, w) {
			runes := []rune(value)
			if len(runes) <= 4 {
				return "****"
			}
			return string(runes[:4]) + "****"
		}
	}
	return value
}

// 获取 claude 配置目录（~/.claude 或 CCENV_CLAUDE_HOME）
func claudeDir() (string, error) {
	if dir := os.Getenv(homeEnvVar); dir != "" {
		if abs, err := filepath.Abs(dir); err == nil {
			return abs, nil
		}
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取 HOME 目录: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// 确保 ~/.claude 目录存在
func ensureClaudeDir() (string, error) {
	dir, err := claudeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("无法创建目录 %s: %w", dir, err)
	}
	return dir, nil
}

// ccenvDirPath 返回 profile 存储目录路径（~/.claude/ccenv/）
func ccenvDirPath() (string, error) {
	base, err := claudeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ccenv"), nil
}

// ensureCcenvDir 确保 ~/.claude/ccenv/ 目录存在并返回路径
func ensureCcenvDir() (string, error) {
	dir, err := ccenvDirPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("无法创建目录 %s: %w", dir, err)
	}
	return dir, nil
}

// profilePath 返回指定 profile 的文件路径
func profilePath(name string) (string, error) {
	dir, err := ccenvDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".profile"), nil
}

func readSettings() (map[string]interface{}, error) {
	dir, err := claudeDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, settingsFile))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("无法读取 %s: %w", settingsFile, err)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("无法解析 %s: %w", settingsFile, err)
	}
	return raw, nil
}

// safeWriteFile 原子写入：先写同目录临时文件，再 Rename 替换目标
func safeWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ccenv-")
	if err != nil {
		return fmt.Errorf("无法创建临时文件: %w", err)
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("同步临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("设置临时文件权限失败: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("无法写入 %s: %w", path, err)
	}
	ok = true
	return nil
}

func getEnvFromSettings(raw map[string]interface{}) map[string]string {
	env := make(map[string]string)
	if envRaw, ok := raw["env"]; ok {
		if envMap, ok := envRaw.(map[string]interface{}); ok {
			for k, v := range envMap {
				if s, ok := v.(string); ok {
					env[k] = s
				} else if v != nil {
					fmt.Fprintf(os.Stderr, "警告: env[%q] 的值不是字符串类型，已跳过\n", k)
				}
			}
		}
	}
	return env
}

// shellEscape 用单引号包裹值并转义内嵌单引号，保证 source 后还原为原始字符串
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellUnescape 解码 shellEscape 产生的单引号格式，仅识别自身写入的格式。
// 编码结构：外层单引号包裹，内嵌单引号编码为 '\''（共 4 字符：结束引号+反斜线+单引号+开始引号）。
// 分割后：首段去掉前导 '，末段去掉尾随 '，各段以字面 ' 拼接。
func shellUnescape(s string) (string, error) {
	if len(s) < 2 || !strings.HasPrefix(s, "'") || !strings.HasSuffix(s, "'") {
		return "", fmt.Errorf("格式错误：期望单引号包裹，got %q", s)
	}
	parts := strings.Split(s, `'\''`)
	parts[0] = strings.TrimPrefix(parts[0], "'")
	last := len(parts) - 1
	parts[last] = strings.TrimSuffix(parts[last], "'")
	return strings.Join(parts, "'"), nil
}

// writeProfile 将 env 写入 ~/.claude/ccenv/<name>.profile（权限 0600）
func writeProfile(name string, env map[string]string) error {
	dir, err := ensureCcenvDir()
	if err != nil {
		return err
	}
	var buf strings.Builder
	buf.WriteString("# ccenv profile: " + name + "\n")
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		buf.WriteString("export " + k + "=" + shellEscape(env[k]) + "\n")
	}
	return safeWriteFile(filepath.Join(dir, name+".profile"), []byte(buf.String()), 0600)
}

// parseProfileData 解析 profile 文件内容，仅识别 ccenv 写入的格式
func parseProfileData(data []byte) (map[string]string, error) {
	env := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rest, ok := strings.CutPrefix(line, "export ")
		if !ok {
			return nil, fmt.Errorf("格式错误行: %q（手动编辑自负其责）", line)
		}
		eq := strings.IndexByte(rest, '=')
		if eq < 0 {
			return nil, fmt.Errorf("格式错误行（缺少 =）: %q", line)
		}
		val, err := shellUnescape(rest[eq+1:])
		if err != nil {
			return nil, fmt.Errorf("无法解析值 %q: %w", line, err)
		}
		env[rest[:eq]] = val
	}
	return env, nil
}

// readProfile 读取并解析指定 profile 文件
func readProfile(name string) (map[string]string, error) {
	path, err := profilePath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile '%s' 不存在", name)
		}
		return nil, fmt.Errorf("无法读取 profile '%s': %w", name, err)
	}
	return parseProfileData(data)
}

// listProfileNames 列出所有 profile 名称（按字母顺序）
func listProfileNames() ([]string, error) {
	dir, err := ccenvDirPath()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("无法读取 profile 目录: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".profile") {
			names = append(names, strings.TrimSuffix(e.Name(), ".profile"))
		}
	}
	sort.Strings(names)
	return names, nil
}

// setActiveProfile 原子更新 ccenv.activate 符号链接指向指定 profile
func setActiveProfile(name string) error {
	base, err := ensureClaudeDir()
	if err != nil {
		return err
	}
	target, err := profilePath(name)
	if err != nil {
		return err
	}
	link := filepath.Join(base, activateFile)
	tmp := link + ".tmp"
	os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return fmt.Errorf("无法创建符号链接: %w", err)
	}
	if err := os.Rename(tmp, link); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("无法更新 %s: %w", activateFile, err)
	}
	return nil
}

// activeProfileName 读取 ccenv.activate 符号链接目标，返回 profile 名
func activeProfileName() string {
	base, err := claudeDir()
	if err != nil {
		return ""
	}
	target, err := os.Readlink(filepath.Join(base, activateFile))
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(target), ".profile")
}

// envWithPrefix 从当前进程环境变量中筛出指定前缀的键值对
func envWithPrefix(prefix string) map[string]string {
	env := make(map[string]string)
	for _, kv := range os.Environ() {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		if key := kv[:eq]; strings.HasPrefix(key, prefix) {
			env[key] = kv[eq+1:]
		}
	}
	return env
}

// 打印 env 内容，自动脱敏
func printEnv(env map[string]string) {
	if len(env) == 0 {
		fmt.Println("  (空)")
		return
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s=%s\n", k, maskValue(k, env[k]))
	}
}

func cmdStatus() error {
	raw, err := readSettings()
	if err != nil {
		return err
	}
	fmt.Println("settings.json 官方 env:")
	printEnv(getEnvFromSettings(raw))

	fmt.Println()
	active := activeProfileName()
	if active == "" {
		fmt.Println("已激活 profile: (无)")
	} else if env, err := readProfile(active); err != nil {
		fmt.Printf("已激活 profile: %s (无法读取: %v)\n", active, err)
	} else {
		fmt.Printf("已激活 profile: %s\n", active)
		printEnv(env)
	}

	fmt.Println()
	fmt.Println("当前环境中的 ANTHROPIC_ 变量:")
	printEnv(envWithPrefix("ANTHROPIC_"))
	return nil
}

func cmdSave(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	raw, err := readSettings()
	if err != nil {
		return err
	}
	env := getEnvFromSettings(raw)
	if err := writeProfile(name, env); err != nil {
		return err
	}
	fmt.Printf("已保存 profile '%s' (%d 个变量)\n", name, len(env))
	return nil
}

func cmdUse(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	env, err := readProfile(name)
	if err != nil {
		return err
	}
	if err := setActiveProfile(name); err != nil {
		return err
	}
	base, _ := claudeDir()
	fmt.Printf("已激活 profile '%s' (%d 个变量)\n", name, len(env))
	fmt.Printf("生效方式: source %s\n", filepath.Join(base, activateFile))
	return nil
}

func cmdList() error {
	names, err := listProfileNames()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Println("没有已保存的 profile")
		return nil
	}
	active := activeProfileName()
	fmt.Println("已保存的 profile:")
	for _, name := range names {
		if name == active {
			fmt.Printf("  %s (已激活)\n", name)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}

func cmdShow(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	env, err := readProfile(name)
	if err != nil {
		return err
	}
	fmt.Printf("profile '%s':\n", name)
	printEnv(env)
	return nil
}

func cmdDelete(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	path, err := profilePath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("profile '%s' 不存在", name)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("无法删除 profile '%s': %w", name, err)
	}
	// 若删除的是当前激活 profile，移除悬空的符号链接
	if activeProfileName() == name {
		base, _ := claudeDir()
		os.Remove(filepath.Join(base, activateFile))
		fmt.Printf("已删除 profile '%s'（已取消激活）\n", name)
	} else {
		fmt.Printf("已删除 profile '%s'\n", name)
	}
	return nil
}

func cmdRename(oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("请指定旧名称和新名称")
	}
	oldPath, err := profilePath(oldName)
	if err != nil {
		return err
	}
	newPath, err := profilePath(newName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("profile '%s' 不存在", oldName)
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("profile '%s' 已存在", newName)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("无法重命名: %w", err)
	}
	// 若重命名的是当前激活 profile，更新符号链接
	if activeProfileName() == oldName {
		if err := setActiveProfile(newName); err != nil {
			fmt.Fprintf(os.Stderr, "警告: 无法更新 %s 符号链接: %v\n", activateFile, err)
		}
	}
	fmt.Printf("已将 profile '%s' 重命名为 '%s'\n", oldName, newName)
	return nil
}

func printUsage() {
	fmt.Printf(`%s - Claude Code 环境变量管理工具

用法:
  %s status              显示 settings.json 官方 env 与已激活 profile
  %s save <name>         将 settings.json 当前 env 保存为 profile
  %s use <name>          激活 profile（更新 ~/.claude/ccenv.activate 符号链接）
  %s list                列出所有 profile（标注已激活）
  %s show <name>         显示 profile 详情
  %s delete <name>       删除某个 profile
  %s rename <old> <new>  重命名 profile
  %s -v                  显示版本信息

Profile 存储: ~/.claude/ccenv/<name>.profile
激活方式:
  source ~/.claude/ccenv/<name>.profile    直接使用某个 profile
  source ~/.claude/ccenv.activate          使用当前激活的 profile（符号链接）
`, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	var err error

	switch cmd {
	case "status":
		err = cmdStatus()
	case "save":
		err = cmdSave(getArg(2))
	case "use":
		err = cmdUse(getArg(2))
	case "list":
		err = cmdList()
	case "show":
		err = cmdShow(getArg(2))
	case "delete":
		err = cmdDelete(getArg(2))
	case "rename":
		err = cmdRename(getArg(2), getArg(3))
	case "-v", "--version":
		fmt.Printf("%s %s (built %s)\n", appName, version, buildTime)
	default:
		fmt.Printf("未知命令: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func getArg(index int) string {
	if index < len(os.Args) {
		return os.Args[index]
	}
	return ""
}
