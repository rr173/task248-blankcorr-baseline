package model

// AgeResult is the outcome of correcting one sample and converting its
// corrected ratio into an age with propagated uncertainty.
type AgeResult struct {
	ID             int64
	RelationID     int64
	SampleID       int64
	BatchID        int64
	CorrectedRatio float64
	CorrectedUnc   float64
	AgeValue       float64 // central age estimate (years)
	AgeUnc         float64 // 1-sigma uncertainty (years)
	AgeLow         float64 // age_value - 2*age_unc
	AgeHigh        float64 // age_value + 2*age_unc
	ExpectedLow    float64
	ExpectedHigh   float64
	AnomalyFlag    bool
	Reason         string
	CreatedAt      int64
}

// SetAnomaly decides whether the result is anomalous given the batch expected
// interval and basic numerical sanity.
func (a *AgeResult) SetAnomaly() {
	a.AnomalyFlag = false
	a.Reason = ""
	if a.CorrectedRatio <= 0 {
		a.AnomalyFlag = true
		a.Reason = "corrected ratio <= 0: blank over-subtraction or negative drift"
		return
	}
	if a.ExpectedLow > 0 && a.ExpectedHigh > 0 {
		// anomaly if the 2-sigma interval does not overlap the expected band
		if a.AgeHigh < a.ExpectedLow || a.AgeLow > a.ExpectedHigh {
			a.AnomalyFlag = true
			a.Reason = "2-sigma age interval outside expected band"
		}
	}
}
