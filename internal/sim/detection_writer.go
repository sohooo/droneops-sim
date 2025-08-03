package sim

import "droneops-sim/internal/enemy"

// DetectionWriter handles enemy detection events.
type DetectionWriter interface {
	WriteDetection(enemy.DetectionRow) error
}

// Optional: Detection writers may support batch mode.
type batchDetectionWriter interface {
	WriteDetections([]enemy.DetectionRow) error
}
