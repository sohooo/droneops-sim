# Contributing Guide

Thank you for your interest in improving **DroneOps-Sim**! This short guide explains how to set up a local development environment and contribute changes.

## Prerequisites

- [Go 1.22+](https://go.dev/doc/install)
- [Make](https://www.gnu.org/software/make/) (for convenience)
- [Docker](https://www.docker.com/) and [Helm](https://helm.sh/) if you plan to build images or run in Kubernetes

## Project Structure

- `cmd/` – main program entry point
- `internal/` – application packages (simulation, telemetry, admin UI, configuration)
- `config/` – default simulation configuration
- `schemas/` – CUE schema for config validation
- `helm/` – Helm chart for Kubernetes deployment

## Setup

1. Clone the repository and download dependencies:
   ```bash
   git clone <repo-url>
   cd droneops-sim
   go mod download
   ```
2. (Optional) Run `make build` to compile the simulator into `./build/droneops-sim`.

## Running Locally

To quickly see telemetry output without a database, run:

```bash
make run
```

By default this loads `config/simulation.yaml`, validates it against `schemas/simulation.cue`, and prints telemetry to STDOUT.

To write to GreptimeDB instead, set the endpoint and table variables:

```bash
export GREPTIMEDB_ENDPOINT=127.0.0.1:4001
export GREPTIMEDB_TABLE=drone_telemetry
export ENEMY_DETECTION_TABLE=enemy_detection
./build/droneops-sim simulate
```

The admin web UI will be available on `http://localhost:8080`.

## Tests

Run the unit tests with:

```bash
make test
```

Configuration files can be validated manually with:

```bash
cue vet config/simulation.yaml schemas/simulation.cue
```

## Docker Image

Build and run the simulator in a container:

```bash
make docker
# or manually
# docker build -t droneops-sim:latest .
# docker run --rm -p 8080:8080 droneops-sim:latest simulate
```

## Kubernetes

A Helm chart is provided under `helm/droneops-sim`. Deploy with:

```bash
cd helm/droneops-sim
helm install droneops-sim .
```

Edit `values.yaml` to customise simulation parameters or resource limits.

## Contributing Workflow

1. Create a new branch for your work.
2. Ensure `go test ./...` passes.
3. Open a pull request describing the change.

We appreciate all improvements, whether in code, tests or documentation!
