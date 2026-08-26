package version

import (
	"context"
	"errors"
	"testing"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

func TestProbeBug08EmptyVersionCannotPublish(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, _ := model.NewBatch("empty", "generic", 1e-4, 1, 0, 0)
	if err := s.CreateBatch(b); err != nil {
		t.Fatal(err)
	}
	if _, err := NewVersioner(s).Publish(context.Background(), b.ID, "empty", "", nil); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("empty publish error = %v, want ErrConflict", err)
	}
	versions, err := s.ListVersions(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Fatalf("empty publish left %d versions", len(versions))
	}
}
