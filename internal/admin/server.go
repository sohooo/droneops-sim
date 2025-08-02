package admin

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
)

type Server struct {
	Sim         *sim.Simulator
	tpl         *template.Template
	mapTpl      *template.Template
	observerTpl *template.Template
}

//go:embed templates/index.html templates/map3d.html templates/observer.html
var content embed.FS

func NewServer(sim *sim.Simulator) *Server {
	tpl := template.Must(template.New("index.html").ParseFS(content, "templates/index.html"))
	mapTpl := template.Must(template.New("map3d.html").ParseFS(content, "templates/map3d.html"))
	observerTpl := template.Must(template.New("observer.html").ParseFS(content, "templates/observer.html"))
	return &Server{Sim: sim, tpl: tpl, mapTpl: mapTpl, observerTpl: observerTpl}
}

func (s *Server) routes() {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/3d", s.handle3D)
	http.HandleFunc("/map-data", s.handleMapData)
	http.HandleFunc("/telemetry", s.handleTelemetry)
	http.HandleFunc("/toggle-chaos", s.handleToggleChaos)
	http.HandleFunc("/launch-drones", s.handleLaunch)
	http.HandleFunc("/fleet-health", s.handleHealth)
	http.HandleFunc("/observer", s.handleObserver)
	http.HandleFunc("/observer/events", s.handleObserverEvents)
	http.HandleFunc("/observer/step", s.handleObserverStep)
	http.HandleFunc("/observer/perspective", s.handleObserverPerspective)
	http.HandleFunc("/observer/command", s.handleObserverCommand)
}

func (s *Server) Start(addr string) error {
	s.routes()
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Chaos        bool
		Fleets       []sim.FleetHealth
		FleetDetails []config.Fleet // Add detailed fleet information
	}{
		Chaos:        s.Sim.Chaos(),
		Fleets:       s.Sim.Health(),
		FleetDetails: s.Sim.GetConfig().Fleets, // Use GetConfig to access fleet details
	}
	s.tpl.Execute(w, data)
}

func (s *Server) handleToggleChaos(w http.ResponseWriter, r *http.Request) {
	state := s.Sim.ToggleChaos()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"chaos": state})
}

func (s *Server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	model := r.URL.Query().Get("model")
	countStr := r.URL.Query().Get("count")
	count, _ := strconv.Atoi(countStr)
	if count <= 0 {
		count = 1
	}
	if model == "" {
		model = "small-fpv"
	}
	s.Sim.LaunchSwarm(model, count)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Sim.Health())
}

func (s *Server) handle3D(w http.ResponseWriter, r *http.Request) {
	s.mapTpl.Execute(w, nil)
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Sim.TelemetrySnapshot())
}

func (s *Server) handleMapData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Sim.MapSnapshot())
}

func (s *Server) handleObserver(w http.ResponseWriter, r *http.Request) {
	s.observerTpl.Execute(w, nil)
}

func (s *Server) handleObserverEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Sim.ObserverEvents())
}

func (s *Server) handleObserverStep(w http.ResponseWriter, r *http.Request) {
	idxStr := r.URL.Query().Get("index")
	idx, _ := strconv.Atoi(idxStr)
	ev, ok := s.Sim.ObserverStep(idx)
	if !ok {
		http.Error(w, "invalid index", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ev)
}

func (s *Server) handleObserverPerspective(w http.ResponseWriter, r *http.Request) {
	drone := r.URL.Query().Get("drone")
	s.Sim.ObserverSetPerspective(drone)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleObserverCommand(w http.ResponseWriter, r *http.Request) {
	cmd := r.URL.Query().Get("cmd")
	s.Sim.ObserverInjectCommand(cmd)
	w.WriteHeader(http.StatusNoContent)
}
