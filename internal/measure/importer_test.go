package measure

import (
	"context"
	"math"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

func TestImportIsIdempotent(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, err := model.NewBatch("batch", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBatch(b); err != nil {
		t.Fatal(err)
	}
	im := NewImporter(s)
	in := model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9, RatioUnc: 0.01}
	first, duplicate, err := im.Import(context.Background(), in)
	if err != nil || duplicate || first.ID == 0 {
		t.Fatalf("first import = id:%v duplicate:%v err:%v", first, duplicate, err)
	}
	second, duplicate, err := im.Import(context.Background(), in)
	if err != nil || !duplicate || second.ID != first.ID {
		t.Fatalf("second import = id:%v duplicate:%v err:%v", second, duplicate, err)
	}
}

// TestImportRejectsNonFiniteValues ensures NaN and +/-Inf are rejected at the
// input boundary and never reach the database, while normal measurements still
// import. Such values are not representable experimental results and would
// propagate into uninterpretable drift, correction and age numbers.
func TestImportRejectsNonFiniteValues(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, err := model.NewBatch("batch", "generic", 1e-4, 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBatch(b); err != nil {
		t.Fatal(err)
	}
	im := NewImporter(s)

	bad := []struct {
		name string
		in   model.MeasurementInput
	}{
		{"NaN ratio", model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: math.NaN()}},
		{"+Inf ratio", model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: math.Inf(1)}},
		{"-Inf ratio", model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: math.Inf(-1)}},
		{"NaN uncertainty", model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9, RatioUnc: math.NaN()}},
		{"+Inf uncertainty", model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "x", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9, RatioUnc: math.Inf(1)}},
		{"NaN certified", model.MeasurementInput{BatchID: b.ID, Kind: model.KindStandard, Material: "std", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9, RatioUnc: 0.01, CertifiedRatio: math.NaN()}},
		{"+Inf certified", model.MeasurementInput{BatchID: b.ID, Kind: model.KindStandard, Material: "std", MeasuredAt: time.UnixMilli(1000), Ratio: 0.9, RatioUnc: 0.01, CertifiedRatio: math.Inf(1)}},
	}
	for _, c := range bad {
		m, dup, err := im.Import(context.Background(), c.in)
		if err == nil {
			t.Fatalf("%s: expected rejection, got measurement %v duplicate=%v", c.name, m, dup)
		}
	}
	// nothing illegal should have been written
	count, err := s.CountStats()
	if err != nil {
		t.Fatal(err)
	}
	if count.Measurements != 0 {
		t.Fatalf("non-finite inputs leaked into the database: %d measurements stored", count.Measurements)
	}

	// a normal measurement must still import fine
	in := model.MeasurementInput{BatchID: b.ID, Kind: model.KindSample, Material: "ok", MeasuredAt: time.UnixMilli(2000), Ratio: 0.9, RatioUnc: 0.01}
	m, _, err := im.Import(context.Background(), in)
	if err != nil || m.ID == 0 {
		t.Fatalf("normal import rejected: m=%v err=%v", m, err)
	}
}
