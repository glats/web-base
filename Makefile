.PHONY: tidy test format lint mock generate dev-bootstrap dev coverage

tidy:
	@go tool -modfile=./tools/go.tool.mod task tidy

test:
	@go tool -modfile=./tools/go.tool.mod task test

format:
	@go tool -modfile=./tools/go.tool.mod task format

lint:
	@go tool -modfile=./tools/go.tool.mod task lint

mock:
	@go tool -modfile=./tools/go.tool.mod task mock

coverage:
	@go tool -modfile=./tools/go.tool.mod task coverage

generate:
	@go tool -modfile=./tools/go.tool.mod task generate

dev-bootstrap:
	@go tool -modfile=./tools/go.tool.mod task dev-bootstrap

dev:
	@go tool -modfile=./tools/go.tool.mod task dev
