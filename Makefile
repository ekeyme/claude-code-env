.PHONY: build install uninstall clean

BINARY = ccenv
INSTALL_DIR = $(HOME)/.local/bin

build:
	go build -o $(BINARY) .

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
