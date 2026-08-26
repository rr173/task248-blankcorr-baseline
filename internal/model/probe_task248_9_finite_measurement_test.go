package model

import (
	"math"
	"testing"
)

func TestProbeBug09MeasurementNumbersMustBeFinite(t *testing.T) {
	inputs := []MeasurementInput{
		{BatchID: 1, Kind: KindSample, MeasuredAt: testTime(), Ratio: math.NaN()},
		{BatchID: 1, Kind: KindSample, MeasuredAt: testTime(), Ratio: math.Inf(1)},
		{BatchID: 1, Kind: KindSample, MeasuredAt: testTime(), Ratio: 1, RatioUnc: math.Inf(1)},
	}
	for i, in := range inputs {
		if err := in.Validate(); err == nil {
			t.Fatalf("input %d with non-finite value was accepted", i)
		}
	}
}
