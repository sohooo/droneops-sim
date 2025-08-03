package sim

import (
	"context"
	"time"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/logging"
	"droneops-sim/internal/telemetry"
)

// Run starts the simulation loop and stops when the context is done.
func (s *Simulator) Run(ctx context.Context) {
	log := logging.FromContext(ctx)
	log.Info("starting simulator", "tick_interval", s.tickInterval)
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick(ctx)
		case <-ctx.Done():
			log.Info("stopping simulator")
			return
		}
	}
}

// tick generates telemetry and writes it.
func (s *Simulator) tick(ctx context.Context) {
	log := logging.FromContext(ctx)
	var batch []telemetry.TelemetryRow
	var detections []enemy.DetectionRow

	s.mu.Lock()
	defer s.mu.Unlock()

	s.messagesSent = 0

	var allDrones []*telemetry.Drone
	for _, f := range s.fleets {
		allDrones = append(allDrones, f.Drones...)
	}
	if s.enemyEng != nil {
		for _, en := range s.enemyEng.Enemies {
			s.enemyPrevPositions[en.ID] = en.Position
		}
		s.enemyEng.Step(allDrones)
	}

	for _, fleet := range s.fleets {
		for _, drone := range fleet.Drones {
			row, ok := s.updateDrone(drone)
			if !ok {
				continue
			}
			if s.chaosMode {
				s.injectChaos(drone, &row)
			}
			batch = append(batch, row)
			detections = append(detections, s.processDetections(&fleet, drone)...)
		}
	}

	s.reassignFollowers()

	// Batch support if writer implements WriteBatch
	if bw, ok := s.writer.(batchWriter); ok {
		if err := bw.WriteBatch(batch); err != nil {
			log.Error("batch write failed", "err", err)
		}
	} else {
		for _, row := range batch {
			if err := s.writer.Write(row); err != nil {
				log.Error("write failed", "drone_id", row.DroneID, "err", err)
			}
		}
	}

	// Write enemy detections if any
	if len(detections) > 0 && s.detectionWriter != nil {
		if bw, ok := s.detectionWriter.(batchDetectionWriter); ok {
			if err := bw.WriteDetections(detections); err != nil {
				log.Error("detection batch write failed", "err", err)
			}
		} else {
			for _, d := range detections {
				if err := s.detectionWriter.WriteDetection(d); err != nil {
					log.Error("detection write failed", "err", err)
				}
			}
		}
	}
}

func (s *Simulator) updateDrone(drone *telemetry.Drone) (telemetry.TelemetryRow, bool) {
	if drone.FollowTarget != nil && (s.rand.Float64() < s.commLoss || drone.Status == telemetry.StatusFailure) {
		s.removeAssignment(drone)
	}
	row := s.teleGen.GenerateTelemetry(drone)
	if s.rand.Float64() < drone.SensorErrorRate {
		row.Lat += s.rand.Float64()*sensorErrorMaxOffset*2 - sensorErrorMaxOffset
		row.Lon += s.rand.Float64()*sensorErrorMaxOffset*2 - sensorErrorMaxOffset
	}
	if s.rand.Float64() < drone.BatteryAnomalyRate {
		drop := s.rand.Float64()*20 + 10
		drone.Battery -= drop
		if drone.Battery < 0 {
			drone.Battery = 0
		}
		row.Battery = drone.Battery
	}
	if s.rand.Float64() < drone.DropoutRate {
		return telemetry.TelemetryRow{}, false
	}
	return row, true
}

func (s *Simulator) injectChaos(drone *telemetry.Drone, row *telemetry.TelemetryRow) {
	if s.rand.Float64() < 0.1 {
		row.Status = telemetry.StatusFailure
		drone.Status = telemetry.StatusFailure
	}
	extra := s.rand.Float64() * 5
	drone.Battery -= extra
	if drone.Battery < 0 {
		drone.Battery = 0
	}
	row.Battery = drone.Battery
}

func (s *Simulator) processDetections(fleet *DroneFleet, drone *telemetry.Drone) []enemy.DetectionRow {
	if s.enemyEng == nil {
		return nil
	}
	var detections []enemy.DetectionRow
	for _, en := range s.enemyEng.Enemies {
		dist := distanceMeters(drone.Position.Lat, drone.Position.Lon, en.Position.Lat, en.Position.Lon)
		if dist > s.detectionRadiusM {
			continue
		}
		conf := 100 * (1 - dist/s.detectionRadiusM)
		conf *= 1 - s.terrainOcclusion
		conf *= 1 - s.weatherImpact
		if s.sensorNoise > 0 {
			conf += s.rand.NormFloat64() * s.sensorNoise * conf
		}
		if conf < 0 {
			conf = 0
		} else if conf > 100 {
			conf = 100
		}
		d := enemy.DetectionRow{
			ClusterID:  s.clusterID,
			DroneID:    drone.ID,
			EnemyID:    en.ID,
			EnemyType:  en.Type,
			Lat:        en.Position.Lat,
			Lon:        en.Position.Lon,
			Alt:        en.Position.Alt,
			Confidence: conf,
			Timestamp:  s.now().UTC(),
		}
		detections = append(detections, d)
		if conf >= s.followConfidence {
			s.assignFollower(fleet, drone, en, conf)
		}
	}
	return detections
}
