package sim

import (
	log "log/slog"
	"math"

	"droneops-sim/internal/enemy"
	"droneops-sim/internal/telemetry"
)

func (s *Simulator) logSwarmEvent(eventType string, drones []string, enemyID string) {
	if len(drones) == 0 || !s.enableSwarmEvents {
		return
	}
	w, ok := s.writer.(SwarmEventWriter)
	if !ok {
		return
	}
	row := telemetry.SwarmEventRow{
		ClusterID: s.clusterID,
		EventType: eventType,
		DroneIDs:  drones,
		EnemyID:   enemyID,
		Timestamp: s.now(),
	}
	if err := w.WriteSwarmEvent(row); err != nil {
		log.Error("swarm event write failed", "err", err)
	}
}

func droneIDSlice(ds []*telemetry.Drone) []string {
	ids := make([]string, len(ds))
	for i, d := range ds {
		ids[i] = d.ID
	}
	return ids
}

func (s *Simulator) sendCommand() bool {
	if s.bandwidthLimit > 0 && s.messagesSent >= s.bandwidthLimit {
		return false
	}
	s.messagesSent++
	if s.rand.Float64() < s.commLoss {
		return false
	}
	return true
}

func (s *Simulator) removeAssignment(drone *telemetry.Drone) {
	enemyID, ok := s.droneAssignments[drone.ID]
	if ok {
		followers := s.enemyFollowers[enemyID]
		nf := followers[:0]
		for _, id := range followers {
			if id != drone.ID {
				nf = append(nf, id)
			}
		}
		if len(nf) == 0 {
			delete(s.enemyFollowers, enemyID)
			delete(s.enemyFollowerTargets, enemyID)
		} else {
			s.enemyFollowers[enemyID] = nf
		}
		delete(s.droneAssignments, drone.ID)
	}
	drone.FollowTarget = nil
}

func (s *Simulator) selectReplacement() *telemetry.Drone {
	var best *telemetry.Drone
	for _, f := range s.fleets {
		for _, d := range f.Drones {
			if d.Status != telemetry.StatusOK || d.FollowTarget != nil {
				continue
			}
			if _, assigned := s.droneAssignments[d.ID]; assigned {
				continue
			}
			if best == nil || d.Battery > best.Battery {
				best = d
			}
		}
	}
	return best
}

// cleanupFollowers removes invalid followers for an enemy and returns remaining active IDs.
func (s *Simulator) cleanupFollowers(enemyID string, followers []string) []string {
	active := followers[:0]
	for _, id := range followers {
		d := s.droneIndex[id]
		if d != nil && d.FollowTarget != nil && d.Status == telemetry.StatusOK {
			active = append(active, id)
			continue
		}
		delete(s.droneAssignments, id)
		if d != nil {
			d.FollowTarget = nil
		}
	}
	return active
}

// selectCandidates finds up to missing replacement drones and reserves their assignments.
func (s *Simulator) selectCandidates(missing int) []*telemetry.Drone {
	var cands []*telemetry.Drone
	for missing > 0 {
		cand := s.selectReplacement()
		if cand == nil || !s.sendCommand() {
			break
		}
		s.droneAssignments[cand.ID] = "" // reserve to avoid reselection
		cands = append(cands, cand)
		missing--
	}
	return cands
}

// filterSendable reserves and returns drones that can receive a command.
func (s *Simulator) filterSendable(cands []*telemetry.Drone) []*telemetry.Drone {
	var selected []*telemetry.Drone
	for _, c := range cands {
		if s.sendCommand() {
			s.droneAssignments[c.ID] = "" // reserve
			selected = append(selected, c)
		}
	}
	return selected
}

// applyAssignments finalizes assignments of candidates to an enemy.
func (s *Simulator) applyAssignments(enemyID string, en *enemy.Enemy, cands []*telemetry.Drone) {
	if len(cands) == 0 {
		return
	}
	pts := s.interceptPoints(en, len(cands))
	for i, d := range cands {
		cp := pts[i]
		d.FollowTarget = &cp
		s.enemyFollowers[enemyID] = append(s.enemyFollowers[enemyID], d.ID)
		s.droneAssignments[d.ID] = enemyID
	}
}

func (s *Simulator) reassignFollowers() {
	for enemyID, followers := range s.enemyFollowers {
		active := s.cleanupFollowers(enemyID, followers)
		if len(active) < len(followers) {
			removed := make([]string, 0, len(followers)-len(active))
			for _, id := range followers {
				found := false
				for _, a := range active {
					if a == id {
						found = true
						break
					}
				}
				if !found {
					removed = append(removed, id)
				}
			}
			s.logSwarmEvent(telemetry.SwarmEventUnassignment, removed, enemyID)
		}
		desired := s.enemyFollowerTargets[enemyID]
		missing := desired - len(active)
		if missing <= 0 {
			if len(active) == 0 {
				delete(s.enemyFollowers, enemyID)
				delete(s.enemyFollowerTargets, enemyID)
			} else {
				s.enemyFollowers[enemyID] = active
			}
			continue
		}
		s.enemyFollowers[enemyID] = active
		en := s.enemyObjects[enemyID]
		cands := s.selectCandidates(missing)
		s.applyAssignments(enemyID, en, cands)
		if len(cands) > 0 {
			s.logSwarmEvent(telemetry.SwarmEventAssignment, droneIDSlice(cands), enemyID)
		}
		if len(s.enemyFollowers[enemyID]) == 0 {
			delete(s.enemyFollowers, enemyID)
			delete(s.enemyFollowerTargets, enemyID)
		}
	}
}

func (s *Simulator) assignFollower(fleet *DroneFleet, detecting *telemetry.Drone, en *enemy.Enemy, conf float64) {
	s.enemyObjects[en.ID] = en
	count, ok := s.swarmResponses[detecting.MovementPattern]
	if !ok {
		count = 0
	}
	if count >= 0 {
		if conf > 90 {
			count++
		}
		switch en.Type {
		case enemy.EnemyVehicle, enemy.EnemyDrone:
			count++
		case enemy.EnemyDecoy:
			if count > 0 {
				count--
			}
		}
		count += s.missionCriticality
	}
	if count == 0 {
		cands := s.filterSendable([]*telemetry.Drone{detecting})
		s.applyAssignments(en.ID, en, cands)
		if len(cands) > 0 {
			s.logSwarmEvent(telemetry.SwarmEventAssignment, droneIDSlice(cands), en.ID)
			s.rebalanceFormation(fleet)
		}
		s.enemyFollowerTargets[en.ID] = len(s.enemyFollowers[en.ID])
		return
	}
	if count < 0 {
		var unassigned []*telemetry.Drone
		for _, d := range fleet.Drones {
			if d.FollowTarget == nil {
				unassigned = append(unassigned, d)
			}
		}
		selected := s.filterSendable(unassigned)
		s.applyAssignments(en.ID, en, selected)
		if len(selected) > 0 {
			s.logSwarmEvent(telemetry.SwarmEventAssignment, droneIDSlice(selected), en.ID)
			s.rebalanceFormation(fleet)
		}
		s.enemyFollowerTargets[en.ID] = len(s.enemyFollowers[en.ID])
		return
	}
	var followers []*telemetry.Drone
	for _, d := range fleet.Drones {
		if d == detecting {
			continue
		}
		if d.FollowTarget == nil {
			followers = append(followers, d)
			if len(followers) >= count {
				break
			}
		}
	}
	if len(followers) == 0 {
		cands := s.filterSendable([]*telemetry.Drone{detecting})
		s.applyAssignments(en.ID, en, cands)
		if len(cands) > 0 {
			s.logSwarmEvent(telemetry.SwarmEventAssignment, droneIDSlice(cands), en.ID)
			s.rebalanceFormation(fleet)
		}
		s.enemyFollowerTargets[en.ID] = len(s.enemyFollowers[en.ID])
		return
	}
	selected := s.filterSendable(followers)
	s.applyAssignments(en.ID, en, selected)
	if len(selected) > 0 {
		s.logSwarmEvent(telemetry.SwarmEventAssignment, droneIDSlice(selected), en.ID)
		s.rebalanceFormation(fleet)
	}
	s.enemyFollowerTargets[en.ID] = len(s.enemyFollowers[en.ID])
}

func (s *Simulator) interceptPoints(en *enemy.Enemy, n int) []telemetry.Position {
	points := make([]telemetry.Position, n)
	target := en.Position
	prev, ok := s.enemyPrevPositions[en.ID]
	velLat := 0.0
	velLon := 0.0
	if ok {
		velLat = target.Lat - prev.Lat
		velLon = target.Lon - prev.Lon
	}
	predicted := telemetry.Position{Lat: target.Lat + velLat*5, Lon: target.Lon + velLon*5, Alt: target.Alt}
	if n == 1 {
		points[0] = predicted
		return points
	}
	norm := math.Hypot(velLat, velLon)
	var perpLat, perpLon float64
	if norm != 0 {
		perpLat = -velLon / norm
		perpLon = velLat / norm
	}
	lateral := interceptLateralMeters
	latStep := lateral / 111000
	lonStep := lateral / (111000 * math.Cos(predicted.Lat*math.Pi/180))
	for i := 0; i < n; i++ {
		offset := float64(i) - float64(n-1)/2
		points[i] = telemetry.Position{
			Lat: predicted.Lat + offset*perpLat*latStep,
			Lon: predicted.Lon + offset*perpLon*lonStep,
			Alt: predicted.Alt,
		}
	}
	return points
}

func (s *Simulator) rebalanceFormation(fleet *DroneFleet) {
	var remaining []*telemetry.Drone
	for _, d := range fleet.Drones {
		if d.FollowTarget == nil {
			remaining = append(remaining, d)
		}
	}
	n := len(remaining)
	if n == 0 {
		return
	}
	region := remaining[0].HomeRegion
	radius := region.RadiusKM * 1000 * 0.5
	for i, d := range remaining {
		angle := float64(i) / float64(n) * 2 * math.Pi
		deltaLat := (radius * math.Cos(angle)) / 111000
		deltaLon := (radius * math.Sin(angle)) / (111000 * math.Cos(region.CenterLat*math.Pi/180))
		d.HomeRegion.CenterLat = region.CenterLat + deltaLat
		d.HomeRegion.CenterLon = region.CenterLon + deltaLon
	}
	s.logSwarmEvent(telemetry.SwarmEventFormationChange, droneIDSlice(remaining), "")
}
