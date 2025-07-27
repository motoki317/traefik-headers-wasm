.PHONY: test
test:
	go test -v -cover ./...

.PHONY: dev
dev:
	docker compose up --build --watch

.PHONY: build-debug
build-debug:
	tinygo build -o plugin.wasm -scheduler=none -target=wasi .

.PHONY: build
build:
	tinygo build -o plugin.wasm -scheduler=none --no-debug -target=wasi .

.PHONY: checks
checks:
	golangci-lint run
