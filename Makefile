.PHONY: test test-unit test-integration test-e2e coverage clean

# 執行所有測試
test:
	go test ./... -v

# 執行單元測試
test-unit:
	go test ./src/internal/domain/... -v -cover

# 執行應用層測試
test-app:
	go test ./src/internal/application/... -v -cover

# 執行集成測試
test-integration:
	go test ./src/internal/infrastructure/... -v -cover

# 執行 E2E 測試
test-e2e:
	go test ./src/test/e2e/... -v

# 生成覆蓋率報告
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 檢查覆蓋率百分比
coverage-check:
	@go test ./... -coverprofile=coverage.out > /dev/null
	@go tool cover -func=coverage.out | grep total | awk '{print "Total Coverage: " $$3}'

# 清理
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# 運行所有 linters
lint:
	golangci-lint run ./...

# 格式化代碼
fmt:
	gofmt -w .
	go mod tidy

# 建置應用
build:
	go build -o bin/app src/cmd/app/main.go

# 執行應用
run:
	go run src/cmd/app/main.go
