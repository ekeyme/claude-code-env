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
	configFile   = "ccenv.config.json"
	settingsFile = "settings.json"
	homeEnvVar   = "CCENV_CLAUDE_HOME"
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

// 获取 claude 配置目录
// 优先使用 CCENV_CLAUDE_HOME 环境变量，否则使用 ~/.claude
func claudeDir() (string, error) {
	// 检查环境变量
	if dir := os.Getenv(homeEnvVar); dir != "" {
		if abs, err := filepath.Abs(dir); err == nil {
			return abs, nil
		}
		// 转换路径失败则使用原始值
		return dir, nil
	}
	// 默认使用 ~/.claude
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取 HOME 目录: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// 确保 claude 目录存在
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

func readSettings() (map[string]interface{}, error) {
	dir, err := claudeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, settingsFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("无法读取 %s: %w", path, err)
	}

	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("无法解析 %s: %w", path, err)
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

// 写入 settings.json，保留所有其他字段
func writeSettings(raw map[string]interface{}) error {
	dir, err := ensureClaudeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, settingsFile)

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("无法序列化 settings: %w", err)
	}
	data = append(data, '\n')

	return safeWriteFile(path, data, 0600)
}

// 从 settings 中获取 env map
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

// 读取 profile 配置
func readProfiles() (map[string]map[string]string, error) {
	dir, err := claudeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, configFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]map[string]string), nil
		}
		return nil, fmt.Errorf("无法读取 %s: %w", path, err)
	}

	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return make(map[string]map[string]string), nil
	}

	var profiles map[string]map[string]string
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("无法解析 %s: %w", path, err)
	}
	return profiles, nil
}

// 写入 profile 配置
func writeProfiles(profiles map[string]map[string]string) error {
	dir, err := ensureClaudeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, configFile)

	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return fmt.Errorf("无法序列化 profiles: %w", err)
	}
	data = append(data, '\n')

	return safeWriteFile(path, data, 0600)
}

// 打印 env 内容，自动脱敏
func printEnv(env map[string]string) {
	if len(env) == 0 {
		fmt.Println("  (空)")
		return
	}
	// 按 key 排序输出
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s=%s\n", k, maskValue(k, env[k]))
	}
}

// cmdStatus 显示当前 settings.json 中的 env
func cmdStatus() error {
	raw, err := readSettings()
	if err != nil {
		return err
	}
	env := getEnvFromSettings(raw)
	fmt.Println("当前 env:")
	printEnv(env)
	return nil
}

// cmdSave 将当前 env 保存为命名 profile
func cmdSave(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	raw, err := readSettings()
	if err != nil {
		return err
	}
	env := getEnvFromSettings(raw)

	profiles, err := readProfiles()
	if err != nil {
		return err
	}
	profiles[name] = env

	if err := writeProfiles(profiles); err != nil {
		return err
	}
	fmt.Printf("已保存 profile '%s' (%d 个变量)\n", name, len(env))
	return nil
}

// cmdUse 应用某个 profile
func cmdUse(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	profiles, err := readProfiles()
	if err != nil {
		return err
	}
	env, ok := profiles[name]
	if !ok {
		return fmt.Errorf("profile '%s' 不存在", name)
	}

	raw, err := readSettings()
	if err != nil {
		return err
	}

	// 将 env 转为 map[string]interface{} 写入 settings
	envIF := make(map[string]interface{})
	for k, v := range env {
		envIF[k] = v
	}
	raw["env"] = envIF

	if err := writeSettings(raw); err != nil {
		return err
	}
	fmt.Printf("已切换到 profile '%s' (%d 个变量)\n", name, len(env))
	return nil
}

// cmdList 列出所有已保存的 profile
func cmdList() error {
	profiles, err := readProfiles()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		fmt.Println("没有已保存的 profile")
		return nil
	}
	fmt.Println("已保存的 profile:")
	names := make([]string, 0, len(profiles))
	for k := range profiles {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  %s (%d 个变量)\n", name, len(profiles[name]))
	}
	return nil
}

// cmdShow 显示某个 profile 的详细内容
func cmdShow(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	profiles, err := readProfiles()
	if err != nil {
		return err
	}
	env, ok := profiles[name]
	if !ok {
		return fmt.Errorf("profile '%s' 不存在", name)
	}
	fmt.Printf("profile '%s':\n", name)
	printEnv(env)
	return nil
}

// cmdDelete 删除某个 profile
func cmdDelete(name string) error {
	if name == "" {
		return fmt.Errorf("请指定 profile 名称")
	}
	profiles, err := readProfiles()
	if err != nil {
		return err
	}
	if _, ok := profiles[name]; !ok {
		return fmt.Errorf("profile '%s' 不存在", name)
	}
	delete(profiles, name)
	if err := writeProfiles(profiles); err != nil {
		return err
	}
	fmt.Printf("已删除 profile '%s'\n", name)
	return nil
}

// cmdRename 重命名某个 profile
func cmdRename(oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("请指定旧名称和新名称")
	}
	profiles, err := readProfiles()
	if err != nil {
		return err
	}
	env, ok := profiles[oldName]
	if !ok {
		return fmt.Errorf("profile '%s' 不存在", oldName)
	}
	if _, ok := profiles[newName]; ok {
		return fmt.Errorf("profile '%s' 已存在", newName)
	}
	profiles[newName] = env
	delete(profiles, oldName)
	if err := writeProfiles(profiles); err != nil {
		return err
	}
	fmt.Printf("已将 profile '%s' 重命名为 '%s'\n", oldName, newName)
	return nil
}

func printUsage() {
	fmt.Printf(`%s - Claude Code 环境变量管理工具

用法:
  %s status              显示当前 env
  %s save <name>         保存当前 env 为 profile
  %s use <name>          应用某个 profile
  %s list                列出所有 profile
  %s show <name>         显示 profile 详情
  %s delete <name>       删除某个 profile
  %s rename <old> <new>  重命名 profile
`, appName, appName, appName, appName, appName, appName, appName, appName)
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

// getArg 安全获取命令行参数
func getArg(index int) string {
	if index < len(os.Args) {
		return os.Args[index]
	}
	return ""
}
