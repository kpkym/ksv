.PHONY: build

build:
	go build -o dist/ksv .
	@if [ -n "$$CUSTOM_USER_BIN_DIR" ]; then \
		cp dist/ksv "$$CUSTOM_USER_BIN_DIR/ksv"; \
		echo "Copied to $$CUSTOM_USER_BIN_DIR/ksv"; \
	fi
