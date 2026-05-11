.PHONY: generate test run-web run-cli
generate:
	@echo "🔧 Generating code..."
	go generate ./...
	@echo "✅ Code generation complete"

test:
	@echo "🧪 Running tests..."
	go test ./... -v
	@echo "✅ Tests complete"

run-web:
	go run ./cmd/web

run-cli:
	go run ./cmd/cli
