
install:
	go mod tidy

build:
	go build

test:
	go test ./...

run:
	./aiapi

clean:
	rm -rf aiapi
