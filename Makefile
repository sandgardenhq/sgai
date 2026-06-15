build: webapp-build lint
	go build -o ./bin/sgai ./cmd/sgai

webapp-build:
	cd cmd/sgai/webapp && bun install && bun run build.ts

webapp-test:
	cd cmd/sgai/webapp && bun install && bun test

webapp-doctor:
	cd cmd/sgai/webapp && bun install && ( npx -y react-doctor@latest . --offline --scope full --blocking error | cat )
	cd cmd/sgai/webapp && score=$$(npx -y react-doctor@latest . --offline --scope full --score | tail -n 1); test "$$score" -ge 100

test: webapp-doctor webapp-test webapp-build
	go test -v ./...
	$(MAKE) lint

webapp-check-deps:
	cd cmd/sgai/webapp && bun outdated

lint:
	GOOS= GOARCH= go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./...

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
