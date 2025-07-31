## Quickstart

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
./build/droneops-sim
```

Docker run:

```bash
docker build -t droneops-sim:latest .
docker run --rm \
    -e GREPTIMEDB_ENDPOINT=127.0.0.1:4001 \
    -e GREPTIMEDB_TABLE=drone_telemetry \
    -e ENEMY_DETECTION_TABLE=enemy_detection \
    droneops-sim:latest
```

