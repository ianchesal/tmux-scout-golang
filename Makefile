.PHONY: build release test clean tag

VERSION := $(shell cat .version 2>/dev/null || echo "dev")

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/tmux-scout .

release:
	GOOS=linux  GOARCH=amd64  go build -ldflags "-X main.version=$(VERSION)" -o bin/tmux-scout-linux-amd64 .
	GOOS=linux  GOARCH=arm64  go build -ldflags "-X main.version=$(VERSION)" -o bin/tmux-scout-linux-arm64 .
	GOOS=darwin GOARCH=amd64  go build -ldflags "-X main.version=$(VERSION)" -o bin/tmux-scout-darwin-amd64 .
	GOOS=darwin GOARCH=arm64  go build -ldflags "-X main.version=$(VERSION)" -o bin/tmux-scout-darwin-arm64 .

test:
	go test ./...

clean:
	rm -rf bin/
	go clean -testcache

tag:
	@set -e; \
	if [ "$(VERSION)" = "dev" ]; then \
		echo "error: .version is missing or empty — update it before tagging"; \
		exit 1; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "error: working tree is not clean — commit or stash changes first"; \
		exit 1; \
	fi; \
	_branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$_branch" != "main" ]; then \
		echo "error: must be on main to tag a release (currently on $$_branch)"; \
		exit 1; \
	fi; \
	if git log origin/main..HEAD --oneline 2>/dev/null | grep -q .; then \
		echo "error: unpushed commits on main — push first"; \
		exit 1; \
	fi; \
	if git tag --list | grep -qx "$(VERSION)"; then \
		echo "error: tag $(VERSION) already exists locally — update .version to a new version number"; \
		exit 1; \
	fi; \
	if git ls-remote --tags origin 2>/dev/null | grep -q "refs/tags/$(VERSION)$$"; then \
		echo "error: tag $(VERSION) already exists on remote — update .version to a new version number"; \
		exit 1; \
	fi; \
	echo "Running tests..."; \
	go test ./...; \
	echo "Tagging $(VERSION) and pushing to trigger GitHub release..."; \
	git tag "$(VERSION)"; \
	git push origin "$(VERSION)"; \
	echo "Done — https://github.com/ianchesal/tmux-scout-golang/releases/tag/$(VERSION)"
