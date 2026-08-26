package match

import (
	"testing"
	"time"

	"task248-blankcorr/internal/model"
)

func TestNearestWithinSkipsTerminalMeasurements(t *testing.T) {
	target := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bad := &model.Measurement{ID: 1, MeasuredAt: target.Add(time.Second), Status: model.MeasContaminated}
	good := &model.Measurement{ID: 2, MeasuredAt: target.Add(2 * time.Second), Status: model.MeasRaw}
	got := NearestWithin([]*model.Measurement{bad, good}, target, time.Minute)
	if got == nil || got.ID != good.ID {
		t.Fatalf("nearest eligible measurement = %#v, want good measurement", got)
	}
}
