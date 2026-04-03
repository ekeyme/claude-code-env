.PHONY: build install uninstall clean help

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
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/
	@echo "已安装到 $(INSTALL_DIR)/$(BINARY)"
	@echo "请确保 $(INSTALL_DIR) 在 PATH 中："
	@echo "  echo 'export PATH=\"$(INSTALL_DIR):\$$PATH\"' >> ~/.bashrc && source ~/.bashrc"

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "已从 $(INSTALL_DIR) 卸载 $(BINARY)"

clean:
	rm -f $(BINARY)
