
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

run:
	./aiapi --port 8887 --data-dir ~/.aiapi-debug

clean:
	rm -rf aiapi frontend/dist/assets
