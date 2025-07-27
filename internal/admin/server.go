package admin

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"droneops-sim/internal/config"
	"droneops-sim/internal/sim"
)

type Server struct {
	Sim *sim.Simulator
	tpl *template.Template
}

func NewServer(sim *sim.Simulator) *Server {
	tpl := template.Must(template.ParseFiles("internal/admin/templates/index.html"))
	return &Server{Sim: sim, tpl: tpl}
}

func (s *Server) routes() {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/toggle-chaos", s.handleToggleChaos)
	http.HandleFunc("/launch-drones", s.handleLaunch)
	http.HandleFunc("/fleet-health", s.handleHealth)
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
