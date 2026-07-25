package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"supervisor-procesos/internal/supervisor"
)

// Controller expone operaciones de consulta y control sobre procesos supervisados.
type Controller interface {
	AllStatus() map[string]supervisor.ProcessStatus
	StatusOf(name string) (supervisor.ProcessStatus, bool)
	StartProcess(name string) error
	StopProcess(name string) error
	RestartProcess(name string) error
}

// Server sirve el API HTTP de control del supervisor.
type Server struct {
	ctrl   Controller
	listen string
}

// NewServer crea el servidor API en la dirección indicada.
func NewServer(ctrl Controller, listen string) *Server {
	return &Server{ctrl: ctrl, listen: listen}
}

// Run escucha peticiones HTTP hasta que el contexto se cancela.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/processes", s.handleProcesses)
	mux.HandleFunc("/api/v1/processes/", s.handleProcessByName)

	httpServer := &http.Server{
		Addr:    s.listen,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("apagado del servidor API: %v", err)
		}
	}()

	log.Printf("servidor API escuchando en http://localhost%s", s.listen)
	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/processes" {
		writeError(w, http.StatusNotFound, "ruta no encontrada")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "método no permitido")
		return
	}

	statuses := s.ctrl.AllStatus()
	list := make([]statusResponse, 0, len(statuses))
	for _, status := range statuses {
		list = append(list, toStatusResponse(status))
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleProcessByName(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/processes/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "nombre de proceso requerido")
		return
	}

	name := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "método no permitido")
			return
		}
		status, ok := s.ctrl.StatusOf(name)
		if !ok {
			writeError(w, http.StatusNotFound, "proceso no encontrado")
			return
		}
		writeJSON(w, http.StatusOK, toStatusResponse(status))
		return
	}

	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "ruta no encontrada")
		return
	}

	action := parts[1]
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "use POST para acciones de control")
		return
	}

	var err error
	switch action {
	case "start":
		err = s.ctrl.StartProcess(name)
	case "stop":
		err = s.ctrl.StopProcess(name)
	case "restart":
		err = s.ctrl.RestartProcess(name)
	default:
		writeError(w, http.StatusNotFound, "acción no soportada: use start, stop o restart")
		return
	}

	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	status, ok := s.ctrl.StatusOf(name)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "acción completada",
			"name":    name,
		})
		return
	}
	writeJSON(w, http.StatusOK, toStatusResponse(status))
}

type statusResponse struct {
	Name                string `json:"name"`
	State               string `json:"state"`
	RestartCount        int    `json:"restart_count"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	LastExitCode        int    `json:"last_exit_code"`
}

func toStatusResponse(status supervisor.ProcessStatus) statusResponse {
	return statusResponse{
		Name:                status.Name,
		State:               string(status.State),
		RestartCount:        status.RestartCount,
		ConsecutiveFailures: status.ConsecutiveFailures,
		LastExitCode:        status.LastExitCode,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("error escribiendo respuesta JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, errorResponse{Error: message})
}
