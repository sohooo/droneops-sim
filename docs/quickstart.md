## Quickstart

Refer to the [Environment Variables](../README.md#environment-variables) table for more configuration options.

### Example Configuration

```yaml
fleets:
  - name: demo
    model: small-fpv
    movement_pattern: patrol
```

### Local Demo (Print Only)

```bash
make build
make run
```

### Write to GreptimeDB

```bash
export GREPTIMEDB_ENDPOINT=127.0.0.1:4001
export GREPTIMEDB_TABLE=drone_telemetry
export ENEMY_DETECTION_TABLE=enemy_detection
export SWARM_EVENT_TABLE=swarm_events
export SIMULATION_STATE_TABLE=simulation_state
export ENABLE_DETECTIONS=true
export ENABLE_SWARM_EVENTS=true
export ENABLE_MOVEMENT_METRICS=true
export ENABLE_SIMULATION_STATE=true
./build/droneops-sim simulate
```

Docker run:

```bash
docker build -t droneops-sim:latest .
docker run --rm \
    -e GREPTIMEDB_ENDPOINT=127.0.0.1:4001 \
    -e GREPTIMEDB_TABLE=drone_telemetry \
    -e ENEMY_DETECTION_TABLE=enemy_detection \
    -e SWARM_EVENT_TABLE=swarm_events \
    -e SIMULATION_STATE_TABLE=simulation_state \
    -e ENABLE_DETECTIONS=true \
    -e ENABLE_SWARM_EVENTS=true \
    -e ENABLE_MOVEMENT_METRICS=true \
    -e ENABLE_SIMULATION_STATE=true \
    droneops-sim:latest simulate
```

To disable specific streams, pass flags such as `--detections=false` or `--simulation-state=false` to the `simulate` command.

![Quickstart Dashboard](images/quickstart-dashboard.png)

