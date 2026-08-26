package httpapi

import (
	"net/http"

	"task248-blankcorr/internal/model"
)

func (s *Server) handleCorrect(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	results, err := s.svc.ComputeAges(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	if results == nil {
		results = []*model.AgeResult{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

// handleRecompute runs match then correct in one call, which is the typical
// "recompute after excluding a bad blank" workflow.
func (s *Server) handleRecompute(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.svc.Match(r.Context(), batchID); err != nil {
		writeError(w, err)
		return
	}
	results, err := s.svc.ComputeAges(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	if results == nil {
		results = []*model.AgeResult{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func (s *Server) handleListResults(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	results, err := s.svc.ListAgeResults(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	if results == nil {
		results = []*model.AgeResult{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}
