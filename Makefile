build: webapp-build lint
	go build -o ./bin/sgai ./cmd/sgai

webapp-build:
	cd cmd/sgai/webapp && bun install && bun run build.ts

webapp-test:
	cd cmd/sgai/webapp && bun install && bun test

test: webapp-test
	go test -v ./...
	$(MAKE) lint

webapp-check-deps:
	cd cmd/sgai/webapp && bun outdated

lint:
	GOOS= GOARCH= go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0 run ./...

install: build
	cp sgai $(HOME)/bin/sgai

deploy: build
	mv -v ./bin/sgai ../sgai/bin/sgai-base
	killall -9 sgai-base

absorb-sgai:
	@find sgai/ -type f ! -name 'README.md' ! -name '.DS_Store' | while read f; do \
		mkdir -p "$$(dirname "cmd/sgai/skel/.sgai/$${f#sgai/}")" || true; \
		mv -v "$$f" "cmd/sgai/skel/.sgai/$${f#sgai/}"; \
	done
