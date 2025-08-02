# Enemy Detection Configuration

This feature simulates hostile objects ("enemies") and reports when drones spot them.

## Purpose

Enemy detections allow you to test how your monitoring stack reacts to potential threats. The simulator
spawns enemy entities across all configured zones and makes them react to nearby drones using evasive,
grouping, and decoy tactics. A detection event is emitted whenever a drone is within range.

## How It Works

1. On startup the simulator creates `enemy_count` enemies in **each** zone defined in `config/simulation.yaml` (default: 3).
2. Each tick the enemies update their position. When drones are nearby they attempt evasive maneuvers,
   may group with other enemies, or spawn decoys to distract pursuers.
3. Every drone checks for enemies within the configured `detection_radius_m` (default: **1000&nbsp;m**). When an enemy is detected an event is generated with a
   confidence value that decreases with distance and is further modified by sensor noise, terrain occlusion and weather impact.
4. Detection events are either printed to STDOUT (print-only mode) or inserted into GreptimeDB.
5. If the detection confidence exceeds `follow_confidence` (see `config/simulation.yaml`), drones may switch to follow mode.
6. The number of drones that follow depends on the base `swarm_responses` setting and may increase with detection confidence, enemy type, or mission criticality.
7. Drones that remain in formation are automatically reassigned to new patrol points to keep coverage balanced.

The event structure is defined in `internal/enemy/types.go` and contains fields such as `enemy_id`,
`enemy_type`, latitude/longitude, confidence and timestamp.

Drone telemetry rows now include a `follow` field indicating whether the drone is actively tracking a target.

## Configuration Options

### Simulation Settings

The following fields in `config/simulation.yaml` control the enemy detection behaviour:

| Field               | Description                                      | Default |
|---------------------|--------------------------------------------------|---------|
| `enemy_count`       | Number of simulated enemies per zone             | `3`     |
| `detection_radius_m`| Radius in meters for enemy detection checks      | `1000`  |
| `sensor_noise`      | Standard deviation of sensor noise (fraction)    | `0`     |
| `terrain_occlusion` | Terrain occlusion factor (0-1)                   | `0`     |
| `weather_impact`    | Weather impact factor (0-1)                      | `0`     |

### GreptimeDB Output

The enemy detection subsystem has a single configuration value when writing to GreptimeDB:

| Environment Variable      | Description                                               | Default            |
|---------------------------|-----------------------------------------------------------|--------------------|
| `ENEMY_DETECTION_TABLE`   | Name of the GreptimeDB table to store detection events.   | `enemy_detection`  |

If the variable is unset the default table name `enemy_detection` is used. In print-only mode this value has
no effect.


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
