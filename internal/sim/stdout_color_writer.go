// ColorStdoutWriter prints human-friendly, colorized telemetry to STDOUT.
package sim

import (
	"fmt"
	"io"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"droneops-sim/internal/config"
	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

const (
	colorReset   = "\x1b[0m"
	colorRed     = "\x1b[31m"
	colorGreen   = "\x1b[32m"
	colorYellow  = "\x1b[33m"
	colorBlue    = "\x1b[34m"
	colorMagenta = "\x1b[35m"
	colorCyan    = "\x1b[36m"
	colorGray    = "\x1b[90m"
)

// ColorStdoutWriter prints telemetry rows using ANSI colors.
type ColorStdoutWriter struct {
	cfg           *config.SimulationConfig
	out           io.Writer
	once          sync.Once
	missionColors map[string]string
	colorIdx      int
}

var missionPalette = []string{colorRed, colorGreen, colorYellow, colorBlue, colorMagenta, colorCyan}

// NewColorStdoutWriter creates a ColorStdoutWriter writing to os.Stdout.
func NewColorStdoutWriter(cfg *config.SimulationConfig) *ColorStdoutWriter {
	return &ColorStdoutWriter{
		cfg:           cfg,
		out:           os.Stdout,
		missionColors: make(map[string]string),
	}
}

func (w *ColorStdoutWriter) getMissionColor(id string) string {
	if c, ok := w.missionColors[id]; ok {
		return c
	}
	c := missionPalette[w.colorIdx%len(missionPalette)]
	w.missionColors[id] = c
	w.colorIdx++
	return c
}

func (w *ColorStdoutWriter) printOverview() {
	if w.cfg == nil {
		return
	}

	fmt.Fprintln(w.out, "Simulation Configuration:")
	tw := tabwriter.NewWriter(w.out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "Follow Confidence:\t%.0f\n", w.cfg.FollowConfidence)
	fmt.Fprintf(tw, "Mission Criticality:\t%s\n", w.cfg.MissionCriticality)
	fmt.Fprintf(tw, "Detection Radius (m):\t%.0f\n", w.cfg.DetectionRadiusM)
	fmt.Fprintf(tw, "Sensor Noise:\t%.2f\n", w.cfg.SensorNoise)
	fmt.Fprintf(tw, "Terrain Occlusion:\t%.2f\n", w.cfg.TerrainOcclusion)
	fmt.Fprintf(tw, "Weather Impact:\t%.2f\n", w.cfg.WeatherImpact)
	fmt.Fprintf(tw, "Communication Loss:\t%.2f\n", w.cfg.CommunicationLoss)
	fmt.Fprintf(tw, "Bandwidth Limit:\t%d\n", w.cfg.BandwidthLimit)
	tw.Flush()

	fmt.Fprintln(w.out, "\nMissions:")
	tw = tabwriter.NewWriter(w.out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID\tName\tObjective\n")
	for _, m := range w.cfg.Missions {
		col := w.getMissionColor(m.ID)
		fmt.Fprintf(tw, "%s%s%s\t%s\t%s\n", col, m.ID, colorReset, m.Name, m.Objective)
	}
	tw.Flush()
	fmt.Fprintln(w.out)
}

// Write outputs a single telemetry row in colorized format.
func (w *ColorStdoutWriter) Write(row telemetry.TelemetryRow) error {
	w.once.Do(w.printOverview)

	mColor := w.getMissionColor(row.MissionID)
	statusColor := colorGreen
	switch row.Status {
	case telemetry.StatusFailure:
		statusColor = colorRed
	case telemetry.StatusLowBattery:
		statusColor = colorYellow
	}

	fmt.Fprintf(w.out, "%s[%s]%s ", colorGray, row.Timestamp.Format(time.RFC3339), colorReset)
	fmt.Fprintf(w.out, "%scluster=%s%s ", colorBlue, row.ClusterID, colorReset)
	fmt.Fprintf(w.out, "%smission=%s%s ", mColor, row.MissionID, colorReset)
	fmt.Fprintf(w.out, "%sdrone=%s%s ", colorWhite(), row.DroneID, colorReset)
	fmt.Fprintf(w.out, "%slat=%.5f%s ", colorGreen, row.Lat, colorReset)
	fmt.Fprintf(w.out, "%slon=%.5f%s ", colorYellow, row.Lon, colorReset)
	fmt.Fprintf(w.out, "%salt=%.1f%s ", colorMagenta, row.Alt, colorReset)
	fmt.Fprintf(w.out, "%sbatt=%.1f%s ", colorCyan, row.Battery, colorReset)
	fmt.Fprintf(w.out, "%spattern=%s%s ", colorBlue, row.MovementPattern, colorReset)
	fmt.Fprintf(w.out, "%sspd=%.1f%s ", colorYellow, row.SpeedMPS, colorReset)
	fmt.Fprintf(w.out, "%shdg=%.1f%s ", colorCyan, row.HeadingDeg, colorReset)
	fmt.Fprintf(w.out, "%sprev=(%.5f,%.5f,%.1f)%s ", colorGray, row.PreviousPosition.Lat, row.PreviousPosition.Lon, row.PreviousPosition.Alt, colorReset)
	fmt.Fprintf(w.out, "%sstatus=%s%s", statusColor, row.Status, colorReset)
	if row.Follow {
		fmt.Fprintf(w.out, " %sfollow%s", colorMagenta, colorReset)
	}
	fmt.Fprintln(w.out)
	return nil
}

func colorWhite() string { return "\x1b[37m" }

// WriteBatch outputs multiple telemetry rows.
func (w *ColorStdoutWriter) WriteBatch(rows []telemetry.TelemetryRow) error {
	for _, r := range rows {
		_ = w.Write(r)
	}
	return nil
}

// WriteDetection prints an enemy detection event to STDOUT.
func (w *ColorStdoutWriter) WriteDetection(d enemy.DetectionRow) error {
	w.once.Do(w.printOverview)
	fmt.Fprintf(w.out, "%s[%s]%s %sDETECTION%s drone=%s enemy=%s type=%s lat=%.5f lon=%.5f alt=%.1f conf=%.2f\n",
		colorGray, d.Timestamp.Format(time.RFC3339), colorReset,
		colorRed, colorReset, d.DroneID, d.EnemyID, d.EnemyType,
		d.Lat, d.Lon, d.Alt, d.Confidence)
	return nil
}

// WriteDetections prints multiple enemy detections.
func (w *ColorStdoutWriter) WriteDetections(rows []enemy.DetectionRow) error {
	for _, d := range rows {
		_ = w.WriteDetection(d)
	}
	return nil
}

// WriteSwarmEvent prints a swarm coordination event to STDOUT.
func (w *ColorStdoutWriter) WriteSwarmEvent(e telemetry.SwarmEventRow) error {
	w.once.Do(w.printOverview)
	fmt.Fprintf(w.out, "%s[%s]%s %sSWARM%s type=%s drones=%v",
		colorGray, e.Timestamp.Format(time.RFC3339), colorReset,
		colorCyan, colorReset, e.EventType, e.DroneIDs)
	if e.EnemyID != "" {
		fmt.Fprintf(w.out, " enemy=%s", e.EnemyID)
	}
	fmt.Fprintln(w.out)
	return nil
}

// WriteSwarmEvents prints multiple swarm events.
func (w *ColorStdoutWriter) WriteSwarmEvents(rows []telemetry.SwarmEventRow) error {
	for _, e := range rows {
		_ = w.WriteSwarmEvent(e)
	}
	return nil
}

// WriteState prints simulation state metrics to STDOUT.
func (w *ColorStdoutWriter) WriteState(row telemetry.SimulationStateRow) error {
	w.once.Do(w.printOverview)
	fmt.Fprintf(w.out, "%s[%s]%s %sSTATE%s comm_loss=%.2f msgs=%d sensor_noise=%.2f weather=%.2f chaos=%t\n",
		colorGray, row.Timestamp.Format(time.RFC3339), colorReset,
		colorBlue, colorReset, row.CommunicationLoss, row.MessagesSent,
		row.SensorNoise, row.WeatherImpact, row.ChaosMode)
	return nil
}

// WriteStates prints multiple simulation state rows.
func (w *ColorStdoutWriter) WriteStates(rows []telemetry.SimulationStateRow) error {
	for _, r := range rows {
		_ = w.WriteState(r)
	}
	return nil
}
