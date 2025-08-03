package sim

// ObserverEvents returns a copy of all recorded mission events.
func (s *Simulator) ObserverEvents() []ObserverEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := make([]ObserverEvent, len(s.observerEvents))
	copy(events, s.observerEvents)
	return events
}

// ObserverStep sets the current event index and returns the event at that position.
func (s *Simulator) ObserverStep(idx int) (ObserverEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx < 0 || idx >= len(s.observerEvents) {
		return ObserverEvent{}, false
	}
	s.observerIdx = idx
	return s.observerEvents[idx], true
}

// ObserverSetPerspective selects a drone to observe.
func (s *Simulator) ObserverSetPerspective(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observerPerspective = id
	s.logObserverEvent("perspective", id)
}

// ObserverPerspective returns the current drone perspective.
func (s *Simulator) ObserverPerspective() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.observerPerspective
}

// ObserverInjectCommand records a scripted command.
func (s *Simulator) ObserverInjectCommand(cmd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logObserverEvent("command", cmd)
}

func (s *Simulator) logObserverEvent(t, details string) {
	s.observerEvents = append(s.observerEvents, ObserverEvent{Timestamp: s.now().UTC(), Type: t, Details: details})
}
