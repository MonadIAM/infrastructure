set quiet
set working-directory := '..'

# Build the Go CLI executable at build/infra.
build:
    go build -o build/infra .

# Run golangci-lint against the Go code.
lint:
    go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run

# Format the Go code in main.go and cli/.
format:
    gofmt -w main.go cli
