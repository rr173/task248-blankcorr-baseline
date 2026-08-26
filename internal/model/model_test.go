package model

import "testing"

func TestBatchValidation(t *testing.T) {
	if _, err := NewBatch("batch", "generic", 1e-4, 1, 10, 100); err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	if _, err := NewBatch("batch", "generic", 0, 1, 10, 100); err == nil {
		t.Fatal("zero decay constant was accepted")
	}
}
