package service

import (
	"context"
	"fmt"

	"task248-blankcorr/internal/model"
)

// SelfCheckResult summarises the health and consistency of a batch. It is the
// backing data for the /api/batches/:id/selfcheck endpoint and the smoke
// test diagnostics.
type SelfCheckResult struct {
	BatchID        int64    `json:"batch_id"`
	Status         string   `json:"status"`
	SampleCount    int      `json:"sample_count"`
	BlankCount     int      `json:"blank_count"`
	StandardCount  int      `json:"standard_count"`
	RelationCount  int      `json:"relation_count"`
	AgeResultCount int      `json:"age_result_count"`
	AnomalyCount   int      `json:"anomaly_count"`
	Sealed         bool     `json:"sealed"`
	Problems       []string `json:"problems"`
}

// SelfCheck validates internal consistency of a batch and reports any
// problems without mutating state.
func (svc *Service) SelfCheck(ctx context.Context, batchID int64) (*SelfCheckResult, error) {
	b, err := svc.store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	active := []string{model.MeasRaw, model.MeasUsable}
	samples, err := svc.store.ListMeasurements(batchID, model.KindSample, active)
	if err != nil {
		return nil, err
	}
	activeSampleIDs := make(map[int64]struct{}, len(samples))
	for _, sample := range samples {
		activeSampleIDs[sample.ID] = struct{}{}
	}
	blanks, err := svc.store.ListMeasurements(batchID, model.KindBlank, active)
	if err != nil {
		return nil, err
	}
	standards, err := svc.store.ListMeasurements(batchID, model.KindStandard, active)
	if err != nil {
		return nil, err
	}
	rels, err := svc.store.ListCorrections(batchID)
	if err != nil {
		return nil, err
	}
	activeRels := rels[:0]
	for _, rel := range rels {
		if _, ok := activeSampleIDs[rel.SampleID]; ok {
			activeRels = append(activeRels, rel)
		}
	}
	rels = activeRels
	ages, err := svc.store.ListAgeResults(batchID)
	if err != nil {
		return nil, err
	}
	activeAges := ages[:0]
	for _, age := range ages {
		if _, ok := activeSampleIDs[age.SampleID]; ok {
			activeAges = append(activeAges, age)
		}
	}
	ages = activeAges

	res := &SelfCheckResult{
		BatchID:        batchID,
		Status:         b.Status,
		SampleCount:    len(samples),
		BlankCount:     len(blanks),
		StandardCount:  len(standards),
		RelationCount:  len(rels),
		AgeResultCount: len(ages),
		Sealed:         b.IsSealed(),
	}

	// consistency problems
	if len(standards) < 1 {
		res.Problems = append(res.Problems, "no standards: drift cannot be modelled")
	}
	if len(blanks) < 1 {
		res.Problems = append(res.Problems, "no blanks: no blank-subtraction term")
	}
	if len(samples) == 0 {
		res.Problems = append(res.Problems, "no samples: nothing to correct")
	}
	if len(rels) != len(samples) {
		res.Problems = append(res.Problems,
			fmt.Sprintf("relation/sample mismatch: %d relations for %d samples", len(rels), len(samples)))
	}
	if len(ages) != len(samples) {
		res.Problems = append(res.Problems,
			fmt.Sprintf("age/sample mismatch: %d age results for %d samples", len(ages), len(samples)))
	}
	for _, a := range ages {
		if a.AnomalyFlag {
			res.AnomalyCount++
		}
	}
	if res.AnomalyCount > 0 {
		res.Problems = append(res.Problems,
			fmt.Sprintf("%d age result(s) flagged anomalous", res.AnomalyCount))
	}
	if b.IsSealed() && len(res.Problems) > 0 {
		// a sealed batch should be internally consistent; surface the mismatch
		res.Problems = append(res.Problems, "sealed batch has consistency problems")
	}
	return res, nil
}
