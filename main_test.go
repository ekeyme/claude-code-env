package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTempDir 创建临时目录并设置 CCENV_CLAUDE_HOME，返回清理函数
func setupTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(homeEnvVar, dir)
	return dir
}

// TestClaudeDir 测试 claudeDir 函数的各种情况
func TestClaudeDir(t *testing.T) {
	oldVal := os.Getenv(homeEnvVar)
	defer func() {
		if oldVal == "" {
			os.Unsetenv(homeEnvVar)
		} else {
			os.Setenv(homeEnvVar, oldVal)
		}
	}()

	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, ".claude")

	tests := []struct {
		name     string
		envValue string
		unset    bool
		want     string
		checkAbs bool
	}{
		{"环境变量设置", "/tmp/test-ccenv", false, "/tmp/test-ccenv", false},
		{"相对路径转绝对", "./test-config", false, "", true},
		{"空字符串回退默认", "", false, defaultPath, false},
		{"未设置使用默认", "", true, defaultPath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				os.Unsetenv(homeEnvVar)
			} else {
				os.Setenv(homeEnvVar, tt.envValue)
			}
			got, err := claudeDir()
			if err != nil {
				t.Fatalf("claudeDir() error: %v", err)
			}
			if tt.checkAbs {
				if !filepath.IsAbs(got) {
					t.Errorf("claudeDir() = %q, expected absolute path", got)
				}
			} else if got != tt.want {
				t.Errorf("claudeDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestMaskValue 测试脱敏函数
func TestMaskValue(t *testing.T) {
	tests := []struct {
		key   string
		value string
		want  string
	}{
		{"API_KEY", "abcdefghij", "abcd****"},
		{"MY_TOKEN", "abc", "****"},
		{"DB_SECRET", "abc", "****"},
		{"PASSWORD", "", "****"},
		{"CREDENTIAL", "my-cred", "my-c****"},
		{"PRIVATE_KEY", "key-data", "key-****"},
		{"token", "value123456", "valu****"},
		{"TOKEN", "value123456", "valu****"},
		{"ToKeN", "value123456", "valu****"},
		{"APP_NAME", "my-app", "my-app"},
		{"BASE_URL", "https://example.com", "https://example.com"},
		{"API_KEY", "密码内容在这里", "密码内容****"},
	}
	for _, tt := range tests {
		got := maskValue(tt.key, tt.value)
		if got != tt.want {
			t.Errorf("maskValue(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.want)
		}
	}
}

// TestShellEscape 测试单引号转义
func TestShellEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"it's", `'it'\''s'`},
		{"$VAR", "'$VAR'"},
		{`back\slash`, `'back\slash'`},
		{"", "''"},
	}
	for _, tt := range tests {
		got := shellEscape(tt.input)
		if got != tt.want {
			t.Errorf("shellEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestShellUnescape 测试 shellEscape 的逆操作
func TestShellUnescape(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"'simple'", "simple", false},
		{"'with space'", "with space", false},
		{`'it'\''s'`, "it's", false},
		{"'$VAR'", "$VAR", false},
		{`'back\slash'`, `back\slash`, false},
		{"''", "", false},
		{"notquoted", "", true},
		{"'unclosed", "", true},
	}
	for _, tt := range tests {
		got, err := shellUnescape(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("shellUnescape(%q) error=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("shellUnescape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestEnvWithPrefix 测试按前缀筛选当前进程环境变量
func TestEnvWithPrefix(t *testing.T) {
	t.Run("只返回匹配前缀的变量", func(t *testing.T) {
		t.Setenv("ANTHROPIC_FOO", "1")
		t.Setenv("ANTHROPIC_BAR", "hello")
		t.Setenv("NOT_ANTHROPIC", "skip")

		got := envWithPrefix("ANTHROPIC_")
		if got["ANTHROPIC_FOO"] != "1" {
			t.Errorf("ANTHROPIC_FOO = %q, want %q", got["ANTHROPIC_FOO"], "1")
		}
		if got["ANTHROPIC_BAR"] != "hello" {
			t.Errorf("ANTHROPIC_BAR = %q, want %q", got["ANTHROPIC_BAR"], "hello")
		}
		if _, ok := got["NOT_ANTHROPIC"]; ok {
			t.Error("NOT_ANTHROPIC 不应被包含")
		}
	})

	t.Run("值中含等号只按首个等号切分", func(t *testing.T) {
		t.Setenv("ANTHROPIC_EQ", "a=b=c")
		got := envWithPrefix("ANTHROPIC_")
		if got["ANTHROPIC_EQ"] != "a=b=c" {
			t.Errorf("ANTHROPIC_EQ = %q, want %q", got["ANTHROPIC_EQ"], "a=b=c")
		}
	})

	t.Run("无匹配返回空 map", func(t *testing.T) {
		got := envWithPrefix("DEFINITELY_NO_SUCH_PREFIX_")
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})
}

// TestParseProfileData 测试 profile 文件解析
func TestParseProfileData(t *testing.T) {
	t.Run("正常解析", func(t *testing.T) {
		data := []byte("# ccenv profile: test\nexport FOO='bar'\nexport BAZ='hello world'\n")
		env, err := parseProfileData(data)
		if err != nil {
			t.Fatalf("parseProfileData error: %v", err)
		}
		if env["FOO"] != "bar" {
			t.Errorf("FOO = %q, want %q", env["FOO"], "bar")
		}
		if env["BAZ"] != "hello world" {
			t.Errorf("BAZ = %q, want %q", env["BAZ"], "hello world")
		}
	})

	t.Run("格式错误行报错", func(t *testing.T) {
		data := []byte("export FOO='bar'\nBAD_LINE\n")
		if _, err := parseProfileData(data); err == nil {
			t.Error("expected error for bad line")
		}
	})
}

// TestWriteAndReadProfile 验证写入和读取 profile 的完整往返
func TestWriteAndReadProfile(t *testing.T) {
	setupTempDir(t)
	env := map[string]string{
		"API_KEY":  "sk-abc123",
		"BASE_URL": "https://example.com",
		"WITH_SPC": "hello world",
	}
	if err := writeProfile("myprof", env); err != nil {
		t.Fatalf("writeProfile error: %v", err)
	}

	// 验证文件权限
	path, _ := profilePath("myprof")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("profile file missing: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("perm = %o, want 0600", perm)
	}

	// 往返读取
	got, err := readProfile("myprof")
	if err != nil {
		t.Fatalf("readProfile error: %v", err)
	}
	for k, want := range env {
		if got[k] != want {
			t.Errorf("key %s: got %q, want %q", k, got[k], want)
		}
	}
}

// TestListProfileNames 验证 profile 列表
func TestListProfileNames(t *testing.T) {
	setupTempDir(t)

	// 空目录
	names, err := listProfileNames()
	if err != nil {
		t.Fatalf("listProfileNames error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty, got %v", names)
	}

	// 写入两个 profile
	writeProfile("beta", map[string]string{"K": "V"})
	writeProfile("alpha", map[string]string{"K": "V"})

	names, _ = listProfileNames()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("listProfileNames = %v, want [alpha beta]", names)
	}
}

// TestSetAndActiveProfileName 验证 symlink 创建与读取
func TestSetAndActiveProfileName(t *testing.T) {
	setupTempDir(t)

	// 无 symlink 时返回空
	if got := activeProfileName(); got != "" {
		t.Errorf("no symlink: got %q, want empty", got)
	}

	writeProfile("work", map[string]string{"K": "V"})
	if err := setActiveProfile("work"); err != nil {
		t.Fatalf("setActiveProfile error: %v", err)
	}
	if got := activeProfileName(); got != "work" {
		t.Errorf("activeProfileName() = %q, want %q", got, "work")
	}

	// 切换到另一个 profile
	writeProfile("home", map[string]string{"K": "V"})
	setActiveProfile("home")
	if got := activeProfileName(); got != "home" {
		t.Errorf("after switch: got %q, want %q", got, "home")
	}
}

// TestCmdUse_WritesSymlinkNotSettings 验证 use 创建 symlink 且不修改 settings.json
func TestCmdUse_WritesSymlinkNotSettings(t *testing.T) {
	dir := setupTempDir(t)

	settingsContent := `{"model":"claude-opus","env":{"OFFICIAL_KEY":"official-value"}}`
	settingsPath := filepath.Join(dir, settingsFile)
	os.WriteFile(settingsPath, []byte(settingsContent), 0600)

	writeProfile("work", map[string]string{
		"ANTHROPIC_BASE_URL":   "https://work.example.com",
		"ANTHROPIC_AUTH_TOKEN": "tok-work",
	})

	if err := cmdUse("work"); err != nil {
		t.Fatalf("cmdUse error: %v", err)
	}

	// settings.json 应完全不变
	got, _ := os.ReadFile(settingsPath)
	if string(got) != settingsContent {
		t.Errorf("settings.json was modified:\ngot:  %s\nwant: %s", got, settingsContent)
	}

	// ccenv.activate 应是指向 work.profile 的 symlink
	if activeProfileName() != "work" {
		t.Error("active profile should be 'work'")
	}
	link := filepath.Join(dir, activateFile)
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("ccenv.activate missing: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("ccenv.activate should be a symlink")
	}
}

// TestCmdUse_ProfileNotFound 验证 profile 不存在时报错且不创建 symlink
func TestCmdUse_ProfileNotFound(t *testing.T) {
	dir := setupTempDir(t)
	writeProfile("other", map[string]string{"K": "V"})

	if err := cmdUse("nonexistent"); err == nil {
		t.Error("expected error for nonexistent profile")
	}
	if _, err := os.Lstat(filepath.Join(dir, activateFile)); !os.IsNotExist(err) {
		t.Error("ccenv.activate should not exist on error")
	}
}

// TestCmdRename_UpdatesSymlink 验证 rename 时自动更新激活状态的 symlink
func TestCmdRename_UpdatesSymlink(t *testing.T) {
	setupTempDir(t)
	writeProfile("old", map[string]string{"K": "V"})
	setActiveProfile("old")

	if err := cmdRename("old", "new"); err != nil {
		t.Fatalf("cmdRename error: %v", err)
	}
	if got := activeProfileName(); got != "new" {
		t.Errorf("after rename: active = %q, want %q", got, "new")
	}
}

// TestCmdDelete_ClearsSymlink 验证删除已激活 profile 时移除 symlink
func TestCmdDelete_ClearsSymlink(t *testing.T) {
	dir := setupTempDir(t)
	writeProfile("todel", map[string]string{"K": "V"})
	setActiveProfile("todel")

	if err := cmdDelete("todel"); err != nil {
		t.Fatalf("cmdDelete error: %v", err)
	}
	// symlink 应被清除
	if activeProfileName() != "" {
		t.Error("symlink should be removed after deleting active profile")
	}
	if _, err := os.Lstat(filepath.Join(dir, activateFile)); !os.IsNotExist(err) {
		t.Error("ccenv.activate should be removed")
	}
}

// TestActivate_SourceProducesExactValues 通过 bash source 验证特殊字符值被正确还原
func TestActivate_SourceProducesExactValues(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	dir := setupTempDir(t)
	env := map[string]string{
		"PLAIN":     "hello",
		"SPACES":    "hello world",
		"DOLLAR":    "$HOME",
		"QUOTE":     "it's fine",
		"BACKSLASH": `back\slash`,
	}
	writeProfile("special", env)
	setActiveProfile("special")

	activatePath := filepath.Join(dir, activateFile)
	for key, want := range env {
		out, err := exec.Command("bash", "-c",
			"source "+activatePath+` && printf '%s' "$`+key+`"`,
		).Output()
		if err != nil {
			t.Errorf("bash source error for %s: %v", key, err)
			continue
		}
		if got := string(out); got != want {
			t.Errorf("key %s: source gave %q, want %q", key, got, want)
		}
	}
}
