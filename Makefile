.PHONY: build test vet verify example
build:
	go build -o bin/gsvc ./cmd/gsvc
test:
	go test -race ./...
vet:
	go vet ./...
verify: test vet build
example: build
	bin/gsvc check --root examples/greeter --verify --strict
