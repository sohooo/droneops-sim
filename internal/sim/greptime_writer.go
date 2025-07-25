package sim

import (
	"context"
	"log"

	"droneops-sim/internal/telemetry"

	greptime "github.com/GreptimeTeam/greptimedb-ingester-go"
	ingesterContext "github.com/GreptimeTeam/greptimedb-ingester-go/context"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table/types"
)

// GreptimeDBWriter writes telemetry to GreptimeDB via the ingester client
type GreptimeDBWriter struct {
	client greptime.Client
	db     string
	table  string
}

// NewGreptimeDBWriter creates a new GreptimeDB writer and auto-creates the table if needed.
func NewGreptimeDBWriter(endpoint, database string) (*GreptimeDBWriter, error) {
	ctx := ingesterContext.NewContext(context.Background())
	client, err := greptime.NewClient(ctx, &greptime.Config{
		Endpoint: endpoint,
	})
	if err != nil {
		return nil, err
	}

	// Auto-create table schema
	ddl := `
CREATE TABLE IF NOT EXISTS drone_telemetry (
  cluster_id STRING TAG,
  drone_id STRING TAG,
  lat DOUBLE,
  lon DOUBLE,
  alt DOUBLE,
  battery DOUBLE,
  status STRING,
  synced_from STRING,
  synced_id STRING,
  synced_at TIMESTAMP,
  ts TIMESTAMP TIME INDEX
) WITH (ttl='30d')
`
	if _, err := client.SQL(ctx, ddl); err != nil {
		return nil, err
	}

	return &GreptimeDBWriter{
		client: client,
		db:     database,
		table:  "drone_telemetry",
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

	ctx := ingesterContext.NewContext(context.Background())

	tbl := table.New(w.table)
	tbl.AddTagColumn("cluster_id", types.StringType, 0)
	tbl.AddTagColumn("drone_id", types.StringType, 0)
	tbl.AddFieldColumn("lat", types.Float64Type)
	tbl.AddFieldColumn("lon", types.Float64Type)
	tbl.AddFieldColumn("alt", types.Float64Type)
	tbl.AddFieldColumn("battery", types.Float64Type)
	tbl.AddFieldColumn("status", types.StringType)
	tbl.AddFieldColumn("synced_from", types.StringType)
	tbl.AddFieldColumn("synced_id", types.StringType)
	tbl.AddFieldColumn("synced_at", types.TimestampType)
	tbl.SetTimeIndex("ts", types.TimestampType)

	for _, r := range rows {
		tbl.AppendTagValue("cluster_id", r.ClusterID)
		tbl.AppendTagValue("drone_id", r.DroneID)
		tbl.AppendFieldValue("lat", r.Lat)
		tbl.AppendFieldValue("lon", r.Lon)
		tbl.AppendFieldValue("alt", r.Alt)
		tbl.AppendFieldValue("battery", r.Battery)
		tbl.AppendFieldValue("status", r.Status)
		tbl.AppendFieldValue("synced_from", r.SyncedFrom)
		tbl.AppendFieldValue("synced_id", r.SyncedID)
		tbl.AppendFieldValue("synced_at", r.SyncedAt)
		tbl.AppendTimeIndex(r.Timestamp)
	}

	if err := w.client.Write(ctx, w.db, []*table.Table{tbl}); err != nil {
		log.Printf("[GreptimeDBWriter] Write failed: %v", err)
		return err
	}

	log.Printf("[GreptimeDBWriter] wrote %d rows", len(rows))
	return nil
}
