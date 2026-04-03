# axon-cost

Initialise the Go module as `github.com/(...)/axon-cost` (library, no main package). Create `go.mod` with dependencies on axon-talk and axon-fact. Add `justfile` with `build`, `test`, and `lint` targets. Add `README.md`, `AGENTS.md`, and `CLAUDE.md` stubs. Verify: `go build ./...` passes on an empty package placeholder.

## Prerequisites

- Go 1.24+
- [just](https://github.com/casey/just)

## Build & Run

```bash
just build
just install
axon-cost --help
```

## Development

```bash
just test   # run tests
just vet    # run go vet
```
