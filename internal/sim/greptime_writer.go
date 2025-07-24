// Writer implementation writing telemetry to GreptimeDB via ORM API
package sim

import (
	"context"
	"log"
	"time"

	greptime "github.com/GreptimeTeam/greptime-go"
	"droneops-sim/internal/telemetry"
)

// GreptimeDBWriter writes telemetry to GreptimeDB.
type GreptimeDBWriter struct {
	client greptime.Client
}

// NewGreptimeDBWriter creates a new GreptimeDB writer and auto-creates the table if needed.
func NewGreptimeDBWriter(endpoint, database string) (*GreptimeDBWriter, error) {
	client, err := greptime.NewClient(&greptime.Config{
		Endpoint: endpoint,
		Database: database,
	})
	if err != nil {
		return nil, err
	}

	// Auto-create table (append-only, no sync state)
	_, err = client.SQL(context.Background(), `
CREATE TABLE IF NOT EXISTS drone_telemetry (
  cluster_id STRING TAG,
  drone_id STRING TAG,
  lat DOUBLE,
  lon DOUBLE,
  alt DOUBLE,
  battery DOUBLE,
  status STRING,
  ts TIMESTAMP TIME INDEX
) WITH (ttl='30d')
`)
	if err != nil {
		return nil, err
	}

	return &GreptimeDBWriter{client: client}, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := w.client.WriteObject(ctx, rows)
	if err != nil {
		log.Printf("[GreptimeDBWriter] WriteObject failed: %v", err)
		return err
	}
	log.Printf("[GreptimeDBWriter] affected rows: %d", resp.GetAffectedRows().GetValue())
	return nil
}