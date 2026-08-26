package model

import "testing"

func TestProbeBug02DraftVersionCannotSeal(t *testing.T) {
	v := NewAgeVersion(1, "draft", "")
	if err := v.Transition(VersionSealed); err == nil {
		t.Fatal("draft version was sealed without publication")
	}
	if v.Status != VersionDraft || !v.SealedAt.IsZero() {
		t.Fatalf("draft version changed after rejected seal: %#v", v)
	}
}
