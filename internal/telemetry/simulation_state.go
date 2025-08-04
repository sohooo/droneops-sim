package telemetry

import "time"

// SimulationStateRow captures per-tick simulator state metrics.
type SimulationStateRow struct {
	ClusterID         string    `json:"cluster_id"`
	CommunicationLoss float64   `json:"communication_loss"`
	MessagesSent      int       `json:"messages_sent"`
	SensorNoise       float64   `json:"sensor_noise"`
	WeatherImpact     float64   `json:"weather_impact"`
	ChaosMode         bool      `json:"chaos_mode"`
	Timestamp         time.Time `json:"ts"`
}
