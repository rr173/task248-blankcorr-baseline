package httpapi

import (
	"net/http"

	"task248-blankcorr/internal/store"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "service": "blankcorr"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.svc.Store().CountStats()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleSelfCheck(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	res, err := s.svc.SelfCheck(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ensure store import is used even if helpers change
var _ = store.Stats{}
