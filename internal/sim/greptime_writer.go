package sim

import (
	"context"
	"encoding/json"
	log "log/slog"
	"strings"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"

	greptime "github.com/GreptimeTeam/greptimedb-ingester-go"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table/types"
)

// GreptimeDBWriter writes telemetry to GreptimeDB via the ingester client
type GreptimeDBWriter struct {
	client         *greptime.Client
	db             string
	table          string
	detectionTable string
	swarmTable     string
	stateTable     string
}

// NewGreptimeDBWriter creates a new GreptimeDB writer.
func NewGreptimeDBWriter(endpoint, database, table string, detectionTable string, swarmTable string, stateTable string) (*GreptimeDBWriter, error) {
	cfg := greptime.NewConfig(endpoint).
		WithPort(4001).
		WithDatabase(database)
	client, err := greptime.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Table creation must be done outside this code (via SQL API or manually).

	if table == "" {
		table = telemetry.TelemetryTableName
	}
	if detectionTable == "" {
		detectionTable = "enemy_detection"
	}
	if swarmTable == "" {
		swarmTable = "swarm_events"
	}
	if stateTable == "" {
		stateTable = "simulation_state"
	}

	return &GreptimeDBWriter{
		client:         client,
		db:             database,
		table:          table,
		detectionTable: detectionTable,
		swarmTable:     swarmTable,
		stateTable:     stateTable,
	}, nil
}

// Write inserts a single telemetry row.
func (w *GreptimeDBWriter) Write(row telemetry.TelemetryRow) error {
	return w.WriteBatch([]telemetry.TelemetryRow{row})
}

// WriteBatch inserts multiple telemetry rows.
func (w *GreptimeDBWriter) WriteBatch(rows []telemetry.TelemetryRow) error {
	if len(rows) == 0 {
		return nil
	}

	ctx := context.Background()

	tbl, err := table.New(w.table)
	if err != nil {
		return err
	}
	tbl.AddTagColumn("cluster_id", types.STRING)
	tbl.AddTagColumn("drone_id", types.STRING)
	tbl.AddTagColumn("mission_id", types.STRING)
	tbl.AddFieldColumn("lat", types.FLOAT64)
	tbl.AddFieldColumn("lon", types.FLOAT64)
	tbl.AddFieldColumn("alt", types.FLOAT64)
	tbl.AddFieldColumn("battery", types.FLOAT64)
	tbl.AddFieldColumn("status", types.STRING)
	tbl.AddFieldColumn("follow", types.BOOLEAN)
	tbl.AddFieldColumn("movement_pattern", types.STRING)
	tbl.AddFieldColumn("speed_mps", types.FLOAT64)
	tbl.AddFieldColumn("heading_deg", types.FLOAT64)
	tbl.AddFieldColumn("previous_position", types.STRING)
	tbl.AddFieldColumn("synced_from", types.STRING)
	tbl.AddFieldColumn("synced_id", types.STRING)
	tbl.AddFieldColumn("synced_at", types.TIMESTAMP_MILLISECOND)
	tbl.AddTimestampColumn("ts", types.TIMESTAMP_MILLISECOND)

	for _, r := range rows {
		prevJSON, _ := json.Marshal(r.PreviousPosition)
		err := tbl.AddRow(
			r.ClusterID,
			r.DroneID,
			r.MissionID,
			r.Lat,
			r.Lon,
			r.Alt,
			r.Battery,
			r.Status,
			r.Follow,
			r.MovementPattern,
			r.SpeedMPS,
			r.HeadingDeg,
			string(prevJSON),
			r.SyncedFrom,
			r.SyncedID,
			r.SyncedAt,
			r.Timestamp,
		)
		if err != nil {
			return err
		}
	}

	_, err = w.client.Write(ctx, tbl)
	if err != nil {
		log.Error("GreptimeDBWriter write failed", "err", err)
		return err
	}

	log.Info("GreptimeDBWriter wrote rows", "count", len(rows))
	return nil
}

// WriteDetection inserts a single enemy detection row.
func (w *GreptimeDBWriter) WriteDetection(d enemy.DetectionRow) error {
	return w.WriteDetections([]enemy.DetectionRow{d})
}

// WriteDetections inserts multiple enemy detection rows.
func (w *GreptimeDBWriter) WriteDetections(rows []enemy.DetectionRow) error {
	if len(rows) == 0 {
		return nil
	}

	ctx := context.Background()

	tbl, err := table.New(w.detectionTable)
	if err != nil {
		return err
	}
	tbl.AddTagColumn("cluster_id", types.STRING)
	tbl.AddTagColumn("drone_id", types.STRING)
	tbl.AddTagColumn("enemy_id", types.STRING)
	tbl.AddTagColumn("enemy_type", types.STRING)
	tbl.AddFieldColumn("lat", types.FLOAT64)
	tbl.AddFieldColumn("lon", types.FLOAT64)
	tbl.AddFieldColumn("alt", types.FLOAT64)
	tbl.AddFieldColumn("drone_lat", types.FLOAT64)
	tbl.AddFieldColumn("drone_lon", types.FLOAT64)
	tbl.AddFieldColumn("drone_alt", types.FLOAT64)
	tbl.AddFieldColumn("distance_m", types.FLOAT64)
	tbl.AddFieldColumn("bearing_deg", types.FLOAT64)
	tbl.AddFieldColumn("enemy_velocity_mps", types.FLOAT64)
	tbl.AddFieldColumn("confidence", types.FLOAT64)
	tbl.AddTimestampColumn("ts", types.TIMESTAMP_MILLISECOND)

	for _, r := range rows {
		err := tbl.AddRow(
			r.ClusterID,
			r.DroneID,
			r.EnemyID,
			string(r.EnemyType),
			r.Lat,
			r.Lon,
			r.Alt,
			r.DroneLat,
			r.DroneLon,
			r.DroneAlt,
			r.DistanceM,
			r.BearingDeg,
			r.EnemyVelMS,
			r.Confidence,
			r.Timestamp,
		)
		if err != nil {
			return err
		}
	}

	_, err = w.client.Write(ctx, tbl)
	if err != nil {
		log.Error("GreptimeDBWriter detection write failed", "err", err)
		return err
	}

	log.Info("GreptimeDBWriter wrote enemy detections", "count", len(rows))
	return nil
}

// WriteSwarmEvent inserts a single swarm event row.
func (w *GreptimeDBWriter) WriteSwarmEvent(e telemetry.SwarmEventRow) error {
	return w.WriteSwarmEvents([]telemetry.SwarmEventRow{e})
}

// WriteSwarmEvents inserts multiple swarm event rows.
func (w *GreptimeDBWriter) WriteSwarmEvents(rows []telemetry.SwarmEventRow) error {
	if len(rows) == 0 {
		return nil
	}

	ctx := context.Background()

	tbl, err := table.New(w.swarmTable)
	if err != nil {
		return err
	}
	tbl.AddTagColumn("cluster_id", types.STRING)
	tbl.AddTagColumn("event_type", types.STRING)
	tbl.AddFieldColumn("drone_ids", types.STRING)
	tbl.AddTagColumn("enemy_id", types.STRING)
	tbl.AddTimestampColumn("ts", types.TIMESTAMP_MILLISECOND)

	for _, r := range rows {
		err := tbl.AddRow(
			r.ClusterID,
			r.EventType,
			strings.Join(r.DroneIDs, ","),
			r.EnemyID,
			r.Timestamp,
		)
		if err != nil {
			return err
		}
	}

	_, err = w.client.Write(ctx, tbl)
	if err != nil {
		log.Error("GreptimeDBWriter swarm event write failed", "err", err)
		return err
	}

	log.Info("GreptimeDBWriter wrote swarm events", "count", len(rows))
	return nil
}

// WriteState inserts a single simulation state row.
func (w *GreptimeDBWriter) WriteState(row telemetry.SimulationStateRow) error {
	return w.WriteStates([]telemetry.SimulationStateRow{row})
}

// WriteStates inserts multiple simulation state rows.
func (w *GreptimeDBWriter) WriteStates(rows []telemetry.SimulationStateRow) error {
	if len(rows) == 0 {
		return nil
	}

	ctx := context.Background()

	tbl, err := table.New(w.stateTable)
	if err != nil {
		return err
	}
	tbl.AddTagColumn("cluster_id", types.STRING)
	tbl.AddFieldColumn("communication_loss", types.FLOAT64)
	tbl.AddFieldColumn("messages_sent", types.INT64)
	tbl.AddFieldColumn("sensor_noise", types.FLOAT64)
	tbl.AddFieldColumn("weather_impact", types.FLOAT64)
	tbl.AddFieldColumn("chaos_mode", types.BOOLEAN)
	tbl.AddTimestampColumn("ts", types.TIMESTAMP_MILLISECOND)

	for _, r := range rows {
		err := tbl.AddRow(
			r.ClusterID,
			r.CommunicationLoss,
			int64(r.MessagesSent),
			r.SensorNoise,
			r.WeatherImpact,
			r.ChaosMode,
			r.Timestamp,
		)
		if err != nil {
			return err
		}
	}

	_, err = w.client.Write(ctx, tbl)
	if err != nil {
		log.Error("GreptimeDBWriter state write failed", "err", err)
		return err
	}
	log.Info("GreptimeDBWriter wrote state rows", "count", len(rows))
	return nil
}
