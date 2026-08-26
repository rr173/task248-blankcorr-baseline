package version

import (
	"context"
	"errors"
	"testing"
	"time"

	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/store"
)

// newSealableStore returns an in-memory store with a single batch ready to
// host versions.
func newSealableStore(t *testing.T) (*store.Store, int64) {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	b, err := model.NewBatch("seal-test", "generic", 1e-4, 1, 900, 1300)
	if err != nil {
		t.Fatalf("new batch: %v", err)
	}
	if err := s.CreateBatch(b); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	return s, b.ID
}

// makeDraftVersion persists a fresh draft version and returns it. It is the
// starting point for the "seal without publishing" attack.
func makeDraftVersion(t *testing.T, s *store.Store, batchID int64) *model.AgeVersion {
	t.Helper()
	v := model.NewAgeVersion(batchID, "draft", "unpublished draft")
	if err := s.CreateVersion(v); err != nil {
		t.Fatalf("create draft version: %v", err)
	}
	return v
}

// makePublishedVersion persists a version that has already completed the
// publish flow, the legitimate pre-seal state.
func makePublishedVersion(t *testing.T, s *store.Store, batchID int64) *model.AgeVersion {
	t.Helper()
	v := makeDraftVersion(t, s, batchID)
	if err := s.UpdateVersionStatus(v.ID, model.VersionPublished); err != nil {
		t.Fatalf("publish version: %v", err)
	}
	v.Status = model.VersionPublished
	return v
}

// TestSealRejectsDraftVersion is the core regression test: sealing a draft
// version (one that never completed the publish flow) must fail, and the
// version's status and sealed_at must be left untouched.
func TestSealRejectsDraftVersion(t *testing.T) {
	s, batchID := newSealableStore(t)
	ver := makeDraftVersion(t, s, batchID)
	verr := NewVersioner(s)

	if _, err := verr.Seal(context.Background(), ver.ID); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("Seal(draft) err = %v, want ErrConflict", err)
	}

	// reload from the store: nothing should have changed. The store renders a
	// zero sealed_at as the Unix epoch, so compare against that rather than
	// Go's zero time.Time.
	got, err := s.GetVersion(ver.ID)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if got.Status != model.VersionDraft {
		t.Errorf("after rejected seal, status = %q, want draft", got.Status)
	}
	if !got.SealedAt.Equal(time.UnixMilli(0)) {
		t.Errorf("after rejected seal, sealed_at = %v, want epoch (unsealed)", got.SealedAt)
	}
	// the owning batch must not have been sealed either
	b, err := s.GetBatch(batchID)
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	if b.IsSealed() {
		t.Errorf("batch sealed after rejected version seal")
	}
}

// TestSealPublishedVersion confirms the happy path still works: a published
// version seals, its sealed_at is stamped, and the owning batch is sealed.
func TestSealPublishedVersion(t *testing.T) {
	s, batchID := newSealableStore(t)
	ver := makePublishedVersion(t, s, batchID)
	verr := NewVersioner(s)
	ctx := context.Background()
	start := time.Now().UTC()

	out, err := verr.Seal(ctx, ver.ID)
	if err != nil {
		t.Fatalf("Seal(published): %v", err)
	}
	if out.Status != model.VersionSealed {
		t.Errorf("returned status = %q, want sealed", out.Status)
	}
	if out.SealedAt.IsZero() || out.SealedAt.Before(start) {
		t.Errorf("returned sealed_at = %v, want a fresh timestamp >= %v", out.SealedAt, start)
	}

	got, err := s.GetVersion(ver.ID)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if got.Status != model.VersionSealed {
		t.Errorf("stored status = %q, want sealed", got.Status)
	}
	if got.SealedAt.IsZero() {
		t.Errorf("stored sealed_at is zero after successful seal")
	}
	b, err := s.GetBatch(batchID)
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	if !b.IsSealed() {
		t.Errorf("batch not sealed after version sealed")
	}

	// re-sealing a sealed version must also be refused
	if _, err := verr.Seal(ctx, ver.ID); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("re-Seal(sealed) err = %v, want ErrConflict", err)
	}
}
