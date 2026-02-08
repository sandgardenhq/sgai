build: webapp-build lint
	go build -o ./bin/sgai ./cmd/sgai

webapp-build:
	cd cmd/sgai/webapp && bun install && bun run build.ts

webapp-test:
	cd cmd/sgai/webapp && bun test

test: webapp-test
	go test -v ./...
	$(MAKE) lint

webapp-check-deps:
	cd cmd/sgai/webapp && bun outdated

lint:
	go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0 run ./...

install: build
	cp sgai $(HOME)/bin/sgai
