build: lint
	go build -o ./bin/sgai ./cmd/sgai

test-go:
	go test -v ./...
	$(MAKE) lint

lint:
	go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0 run ./...

install: build
	cp sgai $(HOME)/bin/sgai
