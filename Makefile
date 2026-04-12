install:
	go install .

build:
	goreleaser build --single-target --snapshot --clean --output dist/kup
	@if [ -n "$$CUSTOM_USER_BIN_DIR" ]; then \
		cp dist/kup "$$CUSTOM_USER_BIN_DIR/kup"; \
		echo "Copied to $$CUSTOM_USER_BIN_DIR/kup"; \
	fi

release:
	goreleaser release --clean

kup:
	go build -o dist/kup .
	@if [ -n "$$CUSTOM_USER_BIN_DIR" ]; then \
		cp dist/kup "$$CUSTOM_USER_BIN_DIR/kup"; \
		echo "Copied to $$CUSTOM_USER_BIN_DIR/kup"; \
	fi
