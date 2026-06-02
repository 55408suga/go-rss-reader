package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// assertGeneratedUUIDv7 fails the test unless id is a non-nil, version-7 UUID,
// matching what the model constructors mint via uuid.NewV7.
func assertGeneratedUUIDv7(t *testing.T, id uuid.UUID) {
	t.Helper()
	if id == uuid.Nil {
		t.Error("id = uuid.Nil, want a generated UUID")
		return
	}
	if got := id.Version(); got != 7 {
		t.Errorf("id.Version() = %d, want 7", got)
	}
}

// assertWithinUTCWindow fails the test unless ts lies within [before, after]
// (inclusive) and carries the UTC location. Used to verify constructor timestamps
// stamped with time.Now().UTC().
func assertWithinUTCWindow(t *testing.T, name string, ts, before, after time.Time) {
	t.Helper()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("%s = %v, want within [%v, %v]", name, ts, before, after)
	}
	if ts.Location() != time.UTC {
		t.Errorf("%s.Location() = %v, want UTC", name, ts.Location())
	}
}
