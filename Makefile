.PHONY: build release test clean

build:
	go build -o bin/tmux-scout .

release:
	GOOS=linux  GOARCH=amd64  go build -o bin/tmux-scout-linux-amd64 .
	GOOS=linux  GOARCH=arm64  go build -o bin/tmux-scout-linux-arm64 .
	GOOS=darwin GOARCH=amd64  go build -o bin/tmux-scout-darwin-amd64 .
	GOOS=darwin GOARCH=arm64  go build -o bin/tmux-scout-darwin-arm64 .

test:
	go test ./...

clean:
	rm -rf bin/
	go clean -testcache
