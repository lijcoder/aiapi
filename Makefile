.PHONY: install hooks build build-all dev-ui test gate check fmt-check vet run clean fmt

# install 顺带启用 git 钩子：全新 clone 走一次 make install 就有门禁兜底。
install: hooks
	go mod tidy

# hooks 启用 .githooks/pre-commit（提交前跑 make check）。core.hooksPath 存在本地 .git/config，
# 不入库，**每个 clone 都要各自执行一次**；重新 clone 后忘记执行 = 门禁静默失效。
hooks:
	git config core.hooksPath .githooks
	@echo "已启用 .githooks/pre-commit：提交前自动跑 make check（跳过用 git commit --no-verify）"

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
# 末尾提示钩子是否已装：没装时本次 check 照样全跑，但提交不会自动触发它（make hooks 开启）。
check: fmt-check vet test
	@git config --get core.hooksPath 2>/dev/null | grep -qx '.githooks' || \
		echo "提示：预提交门禁未启用，执行 make hooks 让提交前自动跑本目标"

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
