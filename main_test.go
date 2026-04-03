package main

import (
	"os"
	"path/filepath"
	"testing"
)

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

	// 获取默认预期值
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
		// 敏感 key - 脱敏
		{"API_KEY", "abcdefghij", "abcd****"},
		{"MY_TOKEN", "abc", "****"},         // 短值（≤4）完全遮蔽
		{"DB_SECRET", "abc", "****"},        // 更短
		{"PASSWORD", "", "****"},            // 空值
		{"CREDENTIAL", "my-cred", "my-c****"},
		{"PRIVATE_KEY", "key-data", "key-****"},
		// 大小写不敏感
		{"token", "value123456", "valu****"},
		{"TOKEN", "value123456", "valu****"},
		{"ToKeN", "value123456", "valu****"},
		// 非敏感 key - 不脱敏
		{"APP_NAME", "my-app", "my-app"},
		{"BASE_URL", "https://example.com", "https://example.com"},
		// Unicode
		{"API_KEY", "密码内容在这里", "密码内容****"},
	}

	for _, tt := range tests {
		got := maskValue(tt.key, tt.value)
		if got != tt.want {
			t.Errorf("maskValue(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.want)
		}
	}
}
