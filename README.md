# DroneOps-Sim

## Overview

**DroneOps-Sim** is a lightweight Go-based simulator that generates realistic drone telemetry for demonstration, testing, and learning purposes.

It supports:

- **Multiple drone fleets** (small FPV, medium UAV, large UAV)
- **Randomized movement patterns** (random walk)
- **Battery drain and failure simulation**
- **Output to GreptimeDB** using its gRPC ORM interface **or** to STDOUT for quick demos

This project was designed to support visualization dashboards (e.g., Grafana Geomap panel) and multi-cluster sync scenarios (mission clusters → command cluster).

For development and contribution guidelines, see [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).

## Purpose

- Provide a realistic demo data source for telemetry pipelines.
- Learn and practice:
  - IoT data modeling for time-series databases
  - GreptimeDB ingestion (gRPC ORM)
  - Kubernetes & Helm deployment
  - Grafana visualization integration
- Serve as a foundation for building more complex simulations (patrol routes, mission-based events).

## Configuration

Detailed configuration options are documented in [docs/configuration.md](docs/configuration.md).

## Schema Validation (schemas/simulation.cue)

Configuration is validated at runtime using CUE:

cue vet config/simulation.yaml schemas/simulation.cue

## Entry Point

### CLI Flags

- `--print-only` → Print telemetry JSON to STDOUT (ignores DB)
- `--config` → Path to YAML config (default: config/simulation.yaml)
- `--schema` → Path to CUE schema (default: schemas/simulation.cue)
- `--tick` → Telemetry tick interval (default: 1s)

### Environment Variables

- `GREPTIMEDB_ENDPOINT` → If set, telemetry is written to this GreptimeDB endpoint
- `GREPTIMEDB_TABLE` → Target table for telemetry (default: drone_telemetry)
- `MISSION_METADATA_TABLE` → Table storing mission metadata (default: mission_metadata)
- `CLUSTER_ID` → Cluster identity tag (default: mission-01)
- `TICK_INTERVAL` → Telemetry tick interval in Go duration format (overrides `--tick`)

## Quickstart

See [docs/quickstart.md](docs/quickstart.md) for step-by-step instructions.

## Examples

### STDOUT output

The simulator first prints the mission metadata and then begins emitting drone
telemetry lines. Example mission record:

```json
{"id":"firewall","name":"Operation: Firewall","objective":"Defend the area from intrusions.","description":"Drones patrol the perimeter to ensure no unauthorized access.","region":{"name":"central-europe","center_lat":48.2,"center_lon":16.4,"radius_km":300}}
```

Example drone telemetry line:

```json
{"cluster_id":"mission-01","drone_id":"recon-swarm-204951-A","mission_id":"","lat":48.19985399217792,"lon":16.399831683018377,"alt":99.89517965510856,"battery":99.5,"status":"ok","synced_from":"","synced_id":"","synced_at":"0001-01-01T00:00:00Z","ts":"2025-07-29T20:49:52.332081195Z"}
```

## Data Flow

```mermaid
flowchart TD
    Start["Start"] -->|"main.go"| A["Parse flags & env"]
    A -->|"config.go"| B["Load config"]
    B -->|"schemas/simulation.cue"| C["Validate with CUE"]
    C -->|"simulator.go"| D["Create simulator"]
    D -->|"Simulator.Run loop"| E["Create drones for fleets"]
    E -->|"generator.go"| F["Generate telemetry"]
    F -->|"types.go"| G["Build TelemetryRow"]
    G -->|"stdout_writer.go"| K["Stdout output"]
    G -->|"greptime_writer.go"| L["GreptimeDB output"]
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
cue vet config/simulation.yaml schemas/simulation.cue
```

Test:

```bash
go test ./... -v
```

## Deployment in Kubernetes

For Helm deployment instructions, see [docs/helm-deployment.md](docs/helm-deployment.md).

## Mission Configuration and Visualization

### Mission Objectives

The `droneops-sim` project supports mission-based drone operations. Missions are defined in the `config/missions.yaml` file and include:

- **ID**: Unique identifier for the mission.
- **Name**: Catchy name matching the mission objective (e.g., "Operation: Firewall").
- **Objective**: The main goal of the mission.
- **Description**: A short story or background for the mission.
- **Region**: The target area for the mission.

Example mission configuration:

```yaml
missions:
  - id: "firewall"
    name: "Operation: Firewall"
    objective: "Defend the area from intrusions."
    description: "Drones patrol the perimeter to ensure no unauthorized access."
    region:
      name: "central-europe"
      center_lat: 48.2
      center_lon: 16.4
      radius_km: 300
```

### Integration with Telemetry

Each drone is associated with a mission via a `mission_id` field in its telemetry. This allows:

- **Mission Visualization**: Display mission objectives and regions in Grafana.
- **Drone Association**: Group and filter drones by mission.

### Grafana Dashboard

The Grafana dashboard integrates mission data to provide:

- **Mission Objectives**: Display the name, objective, and description.
- **Region Visualization**: Overlay mission regions on the Geomap panel.
- **Drone Telemetry**: Filter and group drones by mission.

### Program Logic

On startup, the program:

1. Loads mission data from `config/missions.yaml`.
2. Inserts mission telemetry into stdout or GreptimeDB.
3. Associates drones with missions using the `mission_id` field.

## Admin WebUI

### Features Overview

The Admin WebUI provides a centralized interface for monitoring and managing drone fleets in real-time. It is designed to be lightweight, responsive, and user-friendly.

### Features

- **Fleet Overview**: Displays detailed information about each drone fleet, including model, movement pattern, battery status, and failure rates.
- **Chaos Mode Toggle**: Allows users to enable or disable chaos mode, simulating random failures and unpredictable behavior.
- **Drone Launch Control**: Provides an interface to launch drones for specific missions or operations.
- **Mission Visualization**: Shows mission objectives, regions, and associated drones.
- **Interactive Command Console**: Enables direct interaction with the simulator for advanced operations.

### Access

The Admin WebUI is exposed on port `8080` and can be accessed via a web browser. Ensure the Kubernetes service is correctly configured to route traffic to the Admin WebUI.

### Deployment

The Admin WebUI is included in the Helm chart for the `droneops-sim` project. Follow the steps in the [Deployment in Kubernetes](#deployment-in-kubernetes) section to deploy the simulator and access the Admin WebUI.
