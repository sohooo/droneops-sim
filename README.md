# DroneOps-Sim

## Overview

**DroneOps-Sim** is a lightweight Go-based simulator that generates realistic drone telemetry for demonstration, testing, and learning purposes.

It supports:

- **Multiple drone fleets** (small FPV, medium UAV, large UAV)
- **Randomized movement patterns** (random walk)
- **Battery drain and failure simulation**
- **Output to GreptimeDB** using its gRPC ORM interface **or** to STDOUT for quick demos

This project was designed to support visualization dashboards (e.g., Grafana Geomap panel) and multi-cluster sync scenarios (mission clusters → command cluster).

## Purpose

- Provide a realistic demo data source for telemetry pipelines.
- Learn and practice:
  - IoT data modeling for time-series databases
  - GreptimeDB ingestion (gRPC ORM)
  - Kubernetes & Helm deployment
  - Grafana visualization integration
- Serve as a foundation for building more complex simulations (patrol routes, mission-based events).

## Configuration

### Fleet Configuration (`config/fleet.yaml`)

Defines regions and fleets:
```yaml
regions:
  - name: central-europe
    center_lat: 48.2
    center_lon: 16.4
    radius_km: 300

fleets:
  - name: recon-swarm
    model: small-fpv
    count: 20
    movement_pattern: patrol
    home_region: central-europe
    behavior:
      battery_drain_rate: 0.5
      failure_rate: 0.02
      speed_min_kmh: 50
      speed_max_kmh: 90
```

## Schema Validation (schemas/fleet.cue)

Configuration is validated at runtime using CUE:

cue vet config/fleet.yaml schemas/fleet.cue

## Entry Point

### CLI Flags

-	`--print-only` → Print telemetry JSON to STDOUT (ignores DB)
-	`--config` → Path to YAML config (default: config/fleet.yaml)
-	`--schema` → Path to CUE schema (default: schemas/fleet.cue)

### Environment Variables

-	`GREPTIMEDB_ENDPOINT` → If set, telemetry is written to this GreptimeDB endpoint
-	`CLUSTER_ID` → Cluster identity tag (default: mission-01)

## Quickstart

### Local Demo (Print Only)

```bash
make build
./build/droneops-sim --print-only
```

### Write to GreptimeDB

```bash
export GREPTIMEDB_ENDPOINT=127.0.0.1:4001
./build/droneops-sim
```

Docker run:

```bash
docker build -t droneops-sim:latest .
docker run --rm \
    -e GREPTIMEDB_ENDPOINT=127.0.0.1:4001 \
    droneops-sim:latest
```

## Examples

### STDOUT output

```json
{"cluster_id":"mission-01","drone_id":"recon-swarm-123456-A","lat":48.2023,"lon":16.4098,"alt":100.5,"battery":99.5,"status":"ok","synced_to_json":"[]","ts":"2025-07-23T12:34:56Z"}
```

## Grafana Dashboard (Recommended)

- Use the GreptimeDB data source for Grafana.
- Add a Geomap panel with:
- lat, lon as coordinates
- status, battery as extra fields
- Combine with filters (cluster_id, model) and aggregate views.

## Debugging

- Print mode (--print-only) helps verify telemetry without DB access.
- Logs:
- [Simulator] → telemetry generation
- [GreptimeDBWriter] → DB ingestion results
- Validate config manually:

```bash
cue vet config/fleet.yaml schemas/fleet.cue
```

Test:

```bash
go test ./... -v
```
