# Enemy Detection Configuration

This feature simulates hostile objects ("enemies") and reports when drones spot them.

## Purpose

Enemy detections allow you to test how your monitoring stack reacts to potential threats. The simulator
randomly moves a few enemy entities around the first configured zone and emits a detection event whenever
a drone is within range.

## How It Works

1. On startup the simulator creates three enemies inside the first zone defined in `config/simulation.yaml`.
2. Each tick the enemies take a small random step within that zone.
3. Every drone checks for enemies within **1&nbsp;km**. When an enemy is detected an event is generated with a
   confidence value that decreases with distance.
4. Detection events are either printed to STDOUT (print-only mode) or inserted into GreptimeDB.
5. If the detection confidence exceeds `follow_confidence` (see `config/simulation.yaml`), drones may switch to follow mode.

The event structure is defined in `internal/enemy/types.go` and contains fields such as `enemy_id`,
`enemy_type`, latitude/longitude, confidence and timestamp.

## Configuration Options

The enemy detection subsystem has a single configuration value when writing to GreptimeDB:

| Environment Variable      | Description                                               | Default            |
|---------------------------|-----------------------------------------------------------|--------------------|
| `ENEMY_DETECTION_TABLE`   | Name of the GreptimeDB table to store detection events.   | `enemy_detection`  |

If the variable is unset the default table name `enemy_detection` is used. In print-only mode this value has
no effect.

Currently the number of enemies and detection radius are fixed in code. Future versions may expose these as
configurable parameters.

## Example Event

```json
{
  "cluster_id": "mission-01",
  "drone_id": "recon-swarm-0",
  "enemy_id": "d5b2...",
  "enemy_type": "vehicle",
  "lat": 48.201,
  "lon": 16.403,
  "alt": 0,
  "confidence": 87.5,
  "ts": "2024-06-24T12:00:00Z"
}
```

Use these events to trigger alerts or visualise hostile activity in your dashboards.
