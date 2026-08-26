package httpapi

import (
	"net/http"

	"task248-blankcorr/internal/model"
)

type createBatchReq struct {
	Name          string  `json:"name"`
	SystemType    string  `json:"system_type"`
	Lambda        float64 `json:"lambda"`
	R0            float64 `json:"r0"`
	ExpectedLow   float64 `json:"expected_low"`
	ExpectedHigh  float64 `json:"expected_high"`
}

func (s *Server) handleCreateBatch(w http.ResponseWriter, r *http.Request) {
	var req createBatchReq
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	b, err := s.svc.CreateBatch(r.Context(), req.Name, req.SystemType, req.Lambda, req.R0, req.ExpectedLow, req.ExpectedHigh)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleListBatches(w http.ResponseWriter, r *http.Request) {
	bs, err := s.svc.ListBatches(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if bs == nil {
		bs = []*model.Batch{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"batches": bs})
}

func (s *Server) handleGetBatch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	b, err := s.svc.GetBatch(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleListMeasurements(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	kind := r.URL.Query().Get("kind")
	var statuses []string
	if v := r.URL.Query().Get("status"); v != "" {
		statuses = splitComma(v)
	}
	ms, err := s.svc.ListMeasurements(r.Context(), batchID, kind, statuses)
	if err != nil {
		writeError(w, err)
		return
	}
	if ms == nil {
		ms = []*model.Measurement{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"measurements": ms})
}

// splitComma splits a comma-separated query value into a trimmed slice.
func splitComma(v string) []string {
	out := []string{}
	cur := ""
	for _, c := range v {
		if c == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
