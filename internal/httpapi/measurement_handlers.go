package httpapi

import (
	"net/http"
	"time"

	"task248-blankcorr/internal/model"
)

type importMeasurementReq struct {
	Kind           string  `json:"kind"`
	Material       string  `json:"material"`
	MeasuredAtMS   int64   `json:"measured_at"` // unix milliseconds
	Ratio          float64 `json:"ratio"`
	RatioUnc       float64 `json:"ratio_unc"`
	CertifiedRatio float64 `json:"certified_ratio"`
	SecondaryJSON  string  `json:"secondary_json"`
}

func (s *Server) handleImportMeasurement(w http.ResponseWriter, r *http.Request) {
	batchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req importMeasurementReq
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	in := model.MeasurementInput{
		BatchID:        batchID,
		Kind:           req.Kind,
		Material:       req.Material,
		MeasuredAt:     time.UnixMilli(req.MeasuredAtMS).UTC(),
		Ratio:          req.Ratio,
		RatioUnc:       req.RatioUnc,
		CertifiedRatio: req.CertifiedRatio,
		SecondaryJSON:  req.SecondaryJSON,
	}
	m, dup, err := s.svc.ImportMeasurement(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"measurement": m, "duplicate": dup})
}

type statusReq struct {
	Status string `json:"status"`
}

func (s *Server) handleMeasurementStatus(w http.ResponseWriter, r *http.Request) {
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
	m, err := s.svc.SetMeasurementStatus(r.Context(), id, req.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

type excludeReq struct {
	Reason       string `json:"reason"`
	Contaminated bool   `json:"contaminated"`
}

func (s *Server) handleExclude(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req excludeReq
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	m, err := s.svc.ExcludeMeasurement(r.Context(), id, req.Reason, req.Contaminated)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}
