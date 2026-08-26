package httpapi

import (
	"net/http"

	"task248-blankcorr/internal/model"
)

type publishReq struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req publishReq
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	results, err := s.svc.ListAgeResults(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	ids := make([]int64, 0, len(results))
	for _, a := range results {
		ids = append(ids, a.ID)
	}
	ver, err := s.svc.PublishVersion(r.Context(), batchID, req.Name, req.Note, ids)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ver)
}

func (s *Server) handleSealVersion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	ver, err := s.svc.SealVersion(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ver)
}

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	ver, err := s.svc.GetVersion(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	entries, err := s.svc.Store().ListVersionEntries(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"version": ver, "result_ids": entries})
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	vers, err := s.svc.ListVersions(r.Context(), batchID)
	if err != nil {
		writeError(w, err)
		return
	}
	if vers == nil {
		vers = []*model.AgeVersion{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"versions": vers})
}
