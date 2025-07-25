package sim

import (
	"context"
	"log"

	"droneops-sim/internal/telemetry"

	greptime "github.com/GreptimeTeam/greptimedb-ingester-go"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table/types"
)

// GreptimeDBWriter writes telemetry to GreptimeDB via the ingester client
type GreptimeDBWriter struct {
	client *greptime.Client
	db     string
	table  string
}

// NewGreptimeDBWriter creates a new GreptimeDB writer.
func NewGreptimeDBWriter(endpoint, database, table string) (*GreptimeDBWriter, error) {
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

	return &GreptimeDBWriter{
		client: client,
		db:     database,
		table:  table,
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
	tbl.AddFieldColumn("lat", types.FLOAT64)
	tbl.AddFieldColumn("lon", types.FLOAT64)
	tbl.AddFieldColumn("alt", types.FLOAT64)
	tbl.AddFieldColumn("battery", types.FLOAT64)
	tbl.AddFieldColumn("status", types.STRING)
	tbl.AddFieldColumn("synced_from", types.STRING)
	tbl.AddFieldColumn("synced_id", types.STRING)
	tbl.AddFieldColumn("synced_at", types.TIMESTAMP_MILLISECOND)
	tbl.AddTimestampColumn("ts", types.TIMESTAMP_MILLISECOND)

	for _, r := range rows {
		err := tbl.AddRow(
			r.ClusterID,
			r.DroneID,
			r.Lat,
			r.Lon,
			r.Alt,
			r.Battery,
			r.Status,
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
		log.Printf("[GreptimeDBWriter] Write failed: %v", err)
		return err
	}

	log.Printf("[GreptimeDBWriter] wrote %d rows", len(rows))
	return nil
}
