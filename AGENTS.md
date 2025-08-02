# Agent Guidelines

This repository contains a Go project managed with Go modules and a Makefile.

## Formatting and Linting
- Run `gofmt -w` on any modified Go files before committing.
- If you change dependencies, run `go mod tidy`.
- You may run `go vet ./...` to catch common issues.

## Makefile Commands
- `make build` - compile the simulator to `./build/droneops-sim`.
- `make run` - build and run the simulator with the default config.
- `make docker` - build the Docker image.
- `make clean` - remove build artifacts.
- `make test` - execute unit tests (runs `go test ./...`).

## Tests
Always run `make test` before submitting changes.

## Configuration Validation
Configuration files can be validated with:

```bash
cue vet config/simulation.yaml schemas/simulation.cue
```

