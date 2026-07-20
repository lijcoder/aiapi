
install:
	go mod tidy

build:
	go build

test:
	go test ./...

run:
	./aiapi --port 8887 --data-dir ~/.aiapi-debug

clean:
	rm -rf aiapi
