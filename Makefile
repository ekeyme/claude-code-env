.PHONY: build install uninstall clean help
CLAUDE_DIR = $(HOME)/.claude

BINARY = ccenv
INSTALL_DIR = $(HOME)/.local/bin

# 版本信息（构建时注入）
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_TIME := $(shell date +%Y-%m-%d_%H:%M:%S)
LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.buildTime=$(BUILD_TIME)'

help:
	@echo "Make Targets:"
	@echo "  build      - 构建 ccenv 二进制文件"
	@echo "  install    - 构建并安装到 ~/.local/bin"
	@echo "  uninstall  - 从 ~/.local/bin 卸载 ccenv"
	@echo "  clean      - 清理构建产物"

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/ccenv

install: build
	@mkdir -p $(INSTALL_DIR) $(CLAUDE_DIR)
	cp $(BINARY) $(INSTALL_DIR)/
	cp scripts/ccenv.deactivate $(CLAUDE_DIR)/
	cp scripts/ccenv-claude $(INSTALL_DIR)/ && chmod +x $(INSTALL_DIR)/ccenv-claude
	cp scripts/vscode-claude-wrapper.sh $(INSTALL_DIR)/ && chmod +x $(INSTALL_DIR)/vscode-claude-wrapper.sh
	@echo "已安装到 $(INSTALL_DIR)/$(BINARY)"

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY) $(INSTALL_DIR)/ccenv-claude $(INSTALL_DIR)/vscode-claude-wrapper.sh
	rm -f $(CLAUDE_DIR)/ccenv.deactivate
	@echo "已卸载 $(BINARY)"

clean:
	rm -f $(BINARY)
