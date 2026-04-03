build:
    go build -o bin/axon-cost ./cmd/axon-cost

install: build
    cp bin/axon-cost ~/.local/bin/axon-cost

test:
    go test ./...

vet:
    go vet ./...
