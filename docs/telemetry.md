# Telemetry

Drone telemetry rows include movement metadata to aid analysis and visualization.

## Movement Fields

- `movement_pattern` – current movement strategy (e.g., `patrol`, `point-to-point`, `loiter`).
- `speed_mps` – speed in meters per second derived from the previous position.
- `heading_deg` – bearing from the previous to the current position in degrees.
- `previous_position` – last reported position `{lat, lon, alt}` used for delta calculations.

These fields are emitted alongside existing telemetry attributes and are available in
STDOUT, file logs and GreptimeDB outputs.

```json
{
  "cluster_id": "mission-01",
  "drone_id": "alpha-1",
  "movement_pattern": "patrol",
  "speed_mps": 14.2,
  "heading_deg": 180.0,
  "previous_position": {"lat": 48.2, "lon": 16.4, "alt": 100},
  "lat": 48.3,
  "lon": 16.5,
  "alt": 100,
  "battery": 99.5,
  "status": "ok",
  "ts": "2025-07-29T20:49:52Z"
}
```
