package httpapi

import (
	"net/http"

	"task248-blankcorr/internal/model"
)

func (s *Server) handleMatch(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	rels, err := s.svc.Match(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	if rels == nil {
		rels = []*model.CorrectionRelation{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"relations": rels})
}

func (s *Server) handleListRelations(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	rels, err := s.svc.ListCorrections(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	if rels == nil {
		rels = []*model.CorrectionRelation{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"relations": rels})
}

func (s *Server) handleRelationStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req statusReq
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.svc.SetCorrectionStatus(r.Context(), id, req.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
