// Package httpapi exposes the blank-correction service over HTTP. All routes
// are mounted under the /api prefix. The handlers are thin adapters that
// translate JSON in/out and delegate to the service layer, so the business
// logic lives in one place and is identical to the smoke test path.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/service"
)

// Server holds the service and exposes an http.Handler.
type Server struct {
	svc *service.Service
}

// NewServer constructs an HTTP server around the given service.
func NewServer(svc *service.Service) *Server { return &Server{svc: svc} }

// Handler builds the HTTP router with all /api endpoints registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/stats", s.handleStats)

	mux.HandleFunc("POST /api/batches", s.handleCreateBatch)
	mux.HandleFunc("GET /api/batches", s.handleListBatches)
	mux.HandleFunc("GET /api/batches/{id}", s.handleGetBatch)
	mux.HandleFunc("GET /api/batches/{id}/measurements", s.handleListMeasurements)
	mux.HandleFunc("POST /api/batches/{id}/measurements", s.handleImportMeasurement)
	mux.HandleFunc("GET /api/batches/{id}/relations", s.handleListRelations)
	mux.HandleFunc("POST /api/batches/{id}/match", s.handleMatch)
	mux.HandleFunc("POST /api/batches/{id}/correct", s.handleCorrect)
	mux.HandleFunc("POST /api/batches/{id}/recompute", s.handleRecompute)
	mux.HandleFunc("GET /api/batches/{id}/results", s.handleListResults)
	mux.HandleFunc("GET /api/batches/{id}/versions", s.handleListVersions)
	mux.HandleFunc("GET /api/batches/{id}/selfcheck", s.handleSelfCheck)
	mux.HandleFunc("POST /api/batches/{id}/publish", s.handlePublish)

	mux.HandleFunc("PATCH /api/measurements/{id}/status", s.handleMeasurementStatus)
	mux.HandleFunc("POST /api/measurements/{id}/exclude", s.handleExclude)

	mux.HandleFunc("POST /api/relations/{id}/status", s.handleRelationStatus)

	mux.HandleFunc("GET /api/versions/{id}", s.handleGetVersion)
	mux.HandleFunc("POST /api/versions/{id}/seal", s.handleSealVersion)
	return mux
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError maps domain errors to HTTP status codes.
func writeError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	var ve *model.ValidationError
	switch {
	case errors.Is(err, model.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, model.ErrInvalid), ve != nil:
		code = http.StatusBadRequest
	case errors.Is(err, model.ErrConflict), errors.Is(err, model.ErrSealed):
		code = http.StatusConflict
	}
	writeJSON(w, code, map[string]interface{}{"error": err.Error(), "code": code})
}

// decode reads the request body into v.
func decode(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
