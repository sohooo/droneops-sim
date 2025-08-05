package sim

import (
	"encoding/json"
	"os"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

// FileWriter writes telemetry and detection data to JSONL files.
type FileWriter struct {
	teleFile  *os.File
	detFile   *os.File
	swarmFile *os.File
	stateFile *os.File
	teleEnc   *json.Encoder
	detEnc    *json.Encoder
	swarmEnc  *json.Encoder
	stateEnc  *json.Encoder
}

// NewFileWriter creates a FileWriter. detectionPath, swarmPath, or statePath may be empty to skip those logs.
func NewFileWriter(telemetryPath, detectionPath, swarmPath, statePath string) (*FileWriter, error) {
	tf, err := os.Create(telemetryPath)
	if err != nil {
		return nil, err
	}
	fw := &FileWriter{teleFile: tf, teleEnc: json.NewEncoder(tf)}
	if detectionPath != "" {
		df, err := os.Create(detectionPath)
		if err != nil {
			tf.Close()
			return nil, err
		}
		fw.detFile = df
		fw.detEnc = json.NewEncoder(df)
	}
	if swarmPath != "" {
		sf, err := os.Create(swarmPath)
		if err != nil {
			if fw.detFile != nil {
				fw.detFile.Close()
			}
			tf.Close()
			return nil, err
		}
		fw.swarmFile = sf
		fw.swarmEnc = json.NewEncoder(sf)
	}
	if statePath != "" {
		sf, err := os.Create(statePath)
		if err != nil {
			if fw.detFile != nil {
				fw.detFile.Close()
			}
			if fw.swarmFile != nil {
				fw.swarmFile.Close()
			}
			tf.Close()
			return nil, err
		}
		fw.stateFile = sf
		fw.stateEnc = json.NewEncoder(sf)
	}
	return fw, nil
}

// Write logs a single telemetry row.
func (f *FileWriter) Write(row telemetry.TelemetryRow) error {
	return f.teleEnc.Encode(row)
}

// WriteBatch logs multiple telemetry rows.
func (f *FileWriter) WriteBatch(rows []telemetry.TelemetryRow) error {
	for _, r := range rows {
		if err := f.Write(r); err != nil {
			return err
		}
	}
	return nil
}

// WriteDetection logs a single detection row, if enabled.
func (f *FileWriter) WriteDetection(d enemy.DetectionRow) error {
	if f.detEnc == nil {
		return nil
	}
	return f.detEnc.Encode(d)
}

// WriteDetections logs multiple detection rows.
func (f *FileWriter) WriteDetections(rows []enemy.DetectionRow) error {
	for _, d := range rows {
		if err := f.WriteDetection(d); err != nil {
			return err
		}
	}
	return nil
}

// WriteSwarmEvent logs a single swarm event row, if enabled.
func (f *FileWriter) WriteSwarmEvent(e telemetry.SwarmEventRow) error {
	if f.swarmEnc == nil {
		return nil
	}
	return f.swarmEnc.Encode(e)
}

// WriteSwarmEvents logs multiple swarm events.
func (f *FileWriter) WriteSwarmEvents(rows []telemetry.SwarmEventRow) error {
	for _, r := range rows {
		if err := f.WriteSwarmEvent(r); err != nil {
			return err
		}
	}
	return nil
}

// WriteState logs a simulation state row, if enabled.
func (f *FileWriter) WriteState(row telemetry.SimulationStateRow) error {
	if f.stateEnc == nil {
		return nil
	}
	return f.stateEnc.Encode(row)
}

// WriteStates logs multiple simulation state rows.
func (f *FileWriter) WriteStates(rows []telemetry.SimulationStateRow) error {
	for _, r := range rows {
		if err := f.WriteState(r); err != nil {
			return err
		}
	}
	return nil
}

// WriteMission logs a mission metadata row to the telemetry file.
func (f *FileWriter) WriteMission(row telemetry.MissionRow) error {
	return f.teleEnc.Encode(row)
}

// WriteMissions logs multiple mission metadata rows.
func (f *FileWriter) WriteMissions(rows []telemetry.MissionRow) error {
	for _, r := range rows {
		if err := f.WriteMission(r); err != nil {
			return err
		}
	}
	return nil
}

// Close closes any underlying files.
func (f *FileWriter) Close() error {
	var err error
	if f.teleFile != nil {
		if e := f.teleFile.Close(); e != nil && err == nil {
			err = e
		}
	}
	if f.detFile != nil {
		if e := f.detFile.Close(); e != nil && err == nil {
			err = e
		}
	}
	if f.swarmFile != nil {
		if e := f.swarmFile.Close(); e != nil && err == nil {
			err = e
		}
	}
	if f.stateFile != nil {
		if e := f.stateFile.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}
