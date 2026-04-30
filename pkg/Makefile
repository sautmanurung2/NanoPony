.PHONY: fmt lint test

fmt:
	gofmt -s -w .

lint: fmt
	go vet ./...

test:
	go test -v -race ./...
