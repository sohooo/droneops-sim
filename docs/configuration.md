## Configuration

### Simulation Configuration (`config/simulation.yaml`)

Defines zones, missions, and fleets for the simulation:

```yaml
# Zones define the operational areas for the simulation.
# Each zone includes a name, center coordinates, and a radius.
zones:
  - name: central-europe
    center_lat: 48.2
    center_lon: 16.4
    radius_km: 300

# Missions define the objectives and regions for drone operations.
# Each mission includes an ID, name, objective, description, and associated region.
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
  - id: "recon"
    name: "Operation: Recon"
    objective: "Gather intelligence in the target area."
    description: "Drones perform reconnaissance to collect data on enemy positions."
    region:
      name: "northern-border"
      center_lat: 50.1
      center_lon: 14.4
      radius_km: 200

# Fleets define the drone groups used in the simulation.
# Each fleet includes a name, model, count, movement pattern, home region, and behavior.
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
      sensor_error_rate: 0.01
      dropout_rate: 0.01
      battery_anomaly_rate: 0.01
  - name: transport-squad
    model: medium-uav
    count: 5
    movement_pattern: point-to-point
    home_region: central-europe
    behavior:
      battery_drain_rate: 0.3
      failure_rate: 0.01
      speed_min_kmh: 80
      speed_max_kmh: 140
      sensor_error_rate: 0.01
      dropout_rate: 0.01
      battery_anomaly_rate: 0.01
  - name: heavy-support
    model: large-uav
    count: 2
    movement_pattern: loiter
    home_region: central-europe
    behavior:
      battery_drain_rate: 0.2
      failure_rate: 0.005
      speed_min_kmh: 100
      speed_max_kmh: 180
      sensor_error_rate: 0.01
      dropout_rate: 0.01
      battery_anomaly_rate: 0.01
# Enemy detection settings
enemy_count: 3
detection_radius_m: 1000
# Additional detection factors
sensor_noise: 0.05
terrain_occlusion: 0.1
weather_impact: 0.2
communication_loss: 0.05
bandwidth_limit: 10
# Minimum confidence for drones to begin following detected enemies
follow_confidence: 60

# Overall mission criticality influencing follow aggressiveness
mission_criticality: medium

# Swarm response rules per movement pattern
swarm_responses:
  patrol: 1           # one additional drone follows
  point-to-point: 0   # detecting drone follows
  loiter: 2           # two drones converge
```

`follow_confidence` sets the detection confidence threshold required for a drone
to switch into follow mode (default: `60`). `mission_criticality` (`low`, `medium`, `high`)
adjusts how aggressively the swarm adds followers when a threat is detected.

`enemy_count` controls how many hostile entities are simulated in each zone and `detection_radius_m` sets the detection range in meters for each drone. `sensor_noise`, `terrain_occlusion`, and `weather_impact` modify detection confidence to account for sensor errors and environmental effects.
`communication_loss` introduces the probability that control messages drop or signals fail, and `bandwidth_limit` caps how many commands can be issued per tick, modeling constrained links between drones.

### Enemy Detection

Enemy detection events are stored in GreptimeDB when the `GREPTIMEDB_ENDPOINT` variable is set.
Use `GREPTIMEDB_DATABASE` to select the target database (defaults to `metrics`).
Use `ENEMY_DETECTION_TABLE` to control the table name (default: `enemy_detection`).
See [enemy-detection.md](enemy-detection.md) for more details.

### Telemetry

Control which telemetry streams the simulator emits:

```yaml
telemetry:
  detections: true
  swarm_events: true
  movement_metrics: true
  simulation_state: false
```

- `detections` – output enemy detection events.
- `swarm_events` – log swarm assignment and coordination events.
- `movement_metrics` – include derived speed and heading data.
- `simulation_state` – emit periodic summaries of simulator state.

