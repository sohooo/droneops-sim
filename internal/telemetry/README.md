# Telemetry Module

## Overview
The telemetry module generates realistic drone telemetry records.

### Components
1. **Drone**
   - Holds state for an individual drone (position, battery, status).
2. **TelemetryRow**
   - A database-ready structure with GreptimeDB ORM tags.
3. **Generator**
   - Updates drone state and produces TelemetryRow for each simulation tick.

### Movement Model
- Uses a random walk model.
- Speed and altitude changes depend on drone model type (`small-fpv`, `medium-uav`, `large-uav`).

### Battery Model
- Battery drains per tick based on drone model.
- Drone status transitions:
  - `ok` → normal operation
  - `low_battery` → battery ≤ 20%
  - `failed` → battery ≤ 5%

### Data Flow
1. `Simulator` calls `GenerateTelemetry()` for each drone.
2. `TelemetryRow` is produced and written to GreptimeDB (or printed to STDOUT).
3. Sync agents can later replicate telemetry to command clusters.

### Extensibility
- Add new movement algorithms (e.g., patrol, waypoint missions).
- Integrate environmental effects (wind, GPS noise).
- Add per-fleet behavior overrides (from config).

### Developer Tip
Run with `--print-only` to see telemetry JSON without a DB.