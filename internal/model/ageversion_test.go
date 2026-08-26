package model

import "testing"

// TestVersionStateMachine pins down the age-version lifecycle: a draft must be
// published before it can be sealed, and sealed/superseded versions are terminal.
func TestVersionStateMachine(t *testing.T) {
	v := NewAgeVersion(1, "v1", "")
	if v.Status != VersionDraft {
		t.Fatalf("new version status = %q, want draft", v.Status)
	}
	if !v.SealedAt.IsZero() {
		t.Fatalf("new draft version sealed_at = %v, want zero", v.SealedAt)
	}

	cases := []struct {
		from, to, desc string
		want           bool
	}{
		{VersionDraft, VersionPublished, "draft -> published", true},
		{VersionDraft, VersionSealed, "draft -> sealed (must publish first)", false},
		{VersionDraft, VersionSuperseded, "draft -> superseded", false},
		{VersionDraft, VersionDraft, "draft -> draft", false},
		{VersionPublished, VersionSealed, "published -> sealed", true},
		{VersionPublished, VersionSuperseded, "published -> superseded", true},
		{VersionPublished, VersionPublished, "published -> published", false},
		{VersionPublished, VersionDraft, "published -> draft", false},
		{VersionSealed, VersionPublished, "sealed -> published", false},
		{VersionSealed, VersionSealed, "sealed -> sealed", false},
		{VersionSealed, VersionSuperseded, "sealed -> superseded", false},
		{VersionSuperseded, VersionSealed, "superseded -> sealed", false},
		{VersionSuperseded, VersionPublished, "superseded -> published", false},
	}
	for _, c := range cases {
		v.Status = c.from
		if got := v.CanTransitionTo(c.to); got != c.want {
			t.Errorf("CanTransitionTo(%s -> %s) = %v, want %v (%s)", c.from, c.to, got, c.want, c.desc)
		}
	}
}

// TestVersionTransitionDraftMustNotSeal confirms that calling Transition to
// seal a draft fails and leaves both the status and the sealed timestamp
// untouched.
func TestVersionTransitionDraftMustNotSeal(t *testing.T) {
	v := NewAgeVersion(1, "v1", "")
	before := v.SealedAt
	if err := v.Transition(VersionSealed); err == nil {
		t.Fatal("Transition(draft -> sealed) succeeded; draft must be published before sealing")
	}
	if v.Status != VersionDraft {
		t.Errorf("status changed to %q on rejected seal of draft", v.Status)
	}
	if v.SealedAt != before {
		t.Errorf("sealed_at changed on rejected seal of draft")
	}

	// publishing is allowed and must not stamp sealed_at
	if err := v.Transition(VersionPublished); err != nil {
		t.Fatalf("Transition(draft -> published): %v", err)
	}
	if v.Status != VersionPublished || !v.SealedAt.IsZero() {
		t.Fatalf("after publish: status=%q sealed_at=%v", v.Status, v.SealedAt)
	}

	// once published, sealing is allowed and stamps sealed_at
	if err := v.Transition(VersionSealed); err != nil {
		t.Fatalf("Transition(published -> sealed): %v", err)
	}
	if v.Status != VersionSealed || v.SealedAt.IsZero() {
		t.Fatalf("after seal: status=%q sealed_at=%v", v.Status, v.SealedAt)
	}
}
