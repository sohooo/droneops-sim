package sim

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"droneops-sim/internal/telemetry"
)

type collectWriter struct{ rows []telemetry.TelemetryRow }

func (c *collectWriter) Write(r telemetry.TelemetryRow) error {
	c.rows = append(c.rows, r)
	return nil
}

func TestReplayLog(t *testing.T) {
	rows := []telemetry.TelemetryRow{
		{ClusterID: "c1", DroneID: "d1", Timestamp: time.Unix(0, 0)},
		{ClusterID: "c1", DroneID: "d2", Timestamp: time.Unix(1, 0)},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	cw := &collectWriter{}
	if err := ReplayLog(&buf, cw, 0); err != nil {
		t.Fatalf("ReplayLog: %v", err)
	}
	if len(cw.rows) != len(rows) {
		t.Fatalf("expected %d rows, got %d", len(rows), len(cw.rows))
	}
	for i, r := range rows {
		if cw.rows[i].DroneID != r.DroneID {
			t.Fatalf("row %d mismatch: %+v vs %+v", i, cw.rows[i], r)
		}
	}
}
