package measure

import (
	"context"
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
