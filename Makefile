.PHONY: help help-all build run test lint check install uninstall clean

.DEFAULT_GOAL := help

APP         := $(notdir $(shell go list -m))
TARGET      := target
BINARY      := $(TARGET)/$(APP)
INSTALL_DIR ?= $(HOME)/.local/bin

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "  check      - Run lint + test"
	@echo "  build      - Compile binary to target/"
	@echo "  run        - Build then run (pass args via ARGS=...)"
	@echo "  test       - Run tests"
	@echo "  install    - Copy binary to \$$INSTALL_DIR/"
	@echo "  uninstall  - Remove binary from \$$INSTALL_DIR/"
	@echo ""
	@echo "Run 'make help-all' for the full list, including internal targets."

help-all:
	@echo "Public targets:"
	@echo "  check      - Run lint + test"
	@echo "  build      - Compile binary to target/"
	@echo "  run        - Build then run (pass args via ARGS=...)"
	@echo "  test       - Run tests"
	@echo "  install    - Copy binary to \$$INSTALL_DIR/"
	@echo "  uninstall  - Remove binary from \$$INSTALL_DIR/"
	@echo ""
	@echo "Internal targets (typically called via check):"
	@echo "  lint       - Run go vet + gofmt check"
	@echo "  clean      - Remove compiled binary"

build:
	@mkdir -p $(TARGET)
	@go build -trimpath -o $(BINARY) ./src
	@echo "Built $(BINARY)"

run: build
	@$(BINARY) $(ARGS)

test:
	@go test ./... 2>&1 | tr -d '\r'

lint:
	@unformatted=$$(gofmt -l src); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt'ed:" >&2; \
		echo "$$unformatted" >&2; \
		echo "run: gofmt -w src" >&2; \
		exit 1; \
	fi
	@go vet ./... 2>&1 | tr -d '\r'

check: lint test
	@echo "ALL CHECKS PASSED"

install: build
	@mkdir -p "$(INSTALL_DIR)"
	@tmp="$(INSTALL_DIR)/.$(APP).new.$$$$"; \
	cp "$(BINARY)" "$$tmp"; \
	chmod 755 "$$tmp"; \
	mv -f "$$tmp" "$(INSTALL_DIR)/$(APP)"
	@echo "Installed $(INSTALL_DIR)/$(APP)"
	@local_config="$(TARGET)/config.yaml"; \
	installed_config="$(INSTALL_DIR)/config.yaml"; \
	if [ ! -f "$$local_config" ]; then \
		echo "No $$local_config to copy; $$installed_config is written from the template on first run."; \
	elif [ ! -f "$$installed_config" ] || [ "$${FORCE_CONFIG:-0}" = 1 ]; then \
		cp "$$local_config" "$$installed_config"; \
		echo "Copied settings to $$installed_config"; \
	elif cmp -s "$$local_config" "$$installed_config"; then \
		echo "Settings already match: $$installed_config"; \
	else \
		echo "Kept $$installed_config, it differs from $$local_config"; \
		echo "Overwrite it with: FORCE_CONFIG=1 make install"; \
	fi
	@case ":$$PATH:" in \
	*":$(INSTALL_DIR):"*) ;; \
	*) echo "warning: $(INSTALL_DIR) is not in PATH" >&2 ;; \
	esac

uninstall:
	@if [ -e "$(INSTALL_DIR)/$(APP)" ]; then \
		rm -f "$(INSTALL_DIR)/$(APP)"; \
		echo "Removed $(INSTALL_DIR)/$(APP)"; \
	else \
		echo "Nothing to remove at $(INSTALL_DIR)/$(APP)"; \
	fi
	@echo "Left $(INSTALL_DIR)/config.yaml, datacenters.yaml and the keychain entry alone."
	@echo "Run './$(APP) --logout' before uninstalling to drop the stored credentials."

clean:
	@rm -f "$(BINARY)"
	@echo "Removed $(BINARY)"
	@echo "Kept $(TARGET)/config.yaml and datacenters.yaml (local settings, not build output)."
	@echo "Delete them by hand to start from the template again."
