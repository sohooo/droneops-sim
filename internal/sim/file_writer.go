package sim

import (
	"encoding/json"
	"os"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

// FileWriter writes telemetry and detection data to JSONL files.
type FileWriter struct {
	teleFile *os.File
	detFile  *os.File
	teleEnc  *json.Encoder
	detEnc   *json.Encoder
}

// NewFileWriter creates a FileWriter. detectionPath may be empty to skip detection logs.
func NewFileWriter(telemetryPath, detectionPath string) (*FileWriter, error) {
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
	return err
}
