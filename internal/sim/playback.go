package sim

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"droneops-sim/internal/telemetry"
)

// ReplayLog replays telemetry rows from r to writer. A speed >0 accelerates playback.
// If speed <= 0, no artificial delay is inserted.
func ReplayLog(r io.Reader, writer TelemetryWriter, speed float64) error {
	dec := json.NewDecoder(r)
	var prev time.Time
	for {
		var row telemetry.TelemetryRow
		if err := dec.Decode(&row); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if !prev.IsZero() && speed > 0 {
			diff := row.Timestamp.Sub(prev)
			if speed != 1 {
				diff = time.Duration(float64(diff) / speed)
			}
			if diff > 0 {
				time.Sleep(diff)
			}
		}
		if err := writer.Write(row); err != nil {
			return err
		}
		prev = row.Timestamp
	}
}

// ReplayLogFile opens a file and replays its telemetry rows.
func ReplayLogFile(path string, writer TelemetryWriter, speed float64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return ReplayLog(f, writer, speed)
}
