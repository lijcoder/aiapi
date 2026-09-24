.PHONY: install build build-all dev-ui test check fmt-check vet run clean fmt

install:
	go mod tidy

build:
	go build

build-all:
	cd frontend && npm run build && cd ..
	go build

dev-ui:
	cd frontend && npm run dev

test:
	go test ./...

# check 是提交前必须跑的门禁：格式 + 静态检查 + 测试。
# 架构依赖、权限种子、文档链接/锚点/预算、schema 版本一致性都在 go test 里。
check: fmt-check vet test

fmt-check:
	@unformatted=$$(gofmt -l . | grep -v '^frontend/' || true); \
	if [ -n "$$unformatted" ]; then \
		echo "以下文件未格式化，请执行 make fmt："; echo "$$unformatted"; exit 1; \
	fi

vet:
	go vet ./...

run:
	./aiapi --port 8887 --data-dir ~/.aiapi-debug

clean:
	rm -rf aiapi frontend/dist/assets

fmt:
	go fmt ./...
