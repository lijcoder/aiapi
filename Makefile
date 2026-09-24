.PHONY: install build build-all dev-ui test gate check fmt-check vet run clean fmt

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

# gate 只跑门禁：断言仓库自身的一致性（依赖方向、权限种子、接口文档↔路由、文档链接/预算、
# schema 版本），不验证代码行为。文件以 gate_ 开头、函数以 TestGate 开头。
gate:
	go test -run '^TestGate' ./...

# check 是提交前必须跑的：格式 + 静态检查 + 全部测试（含门禁，所以不再单独跑 gate）。
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
