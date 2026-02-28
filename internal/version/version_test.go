package version

import (
	"testing"
)

func TestString_dev(t *testing.T) {
	// Default values (no ldflags injection)
	Version = "dev"
	Commit = "unknown"
	Date = "unknown"

	got := String()
	if got != "dev" {
		t.Errorf("String() = %q, want %q", got, "dev")
	}
}

func TestString_release(t *testing.T) {
	Version = "1.0.0"
	Commit = "abc1234"
	Date = "2026-01-01T00:00:00Z"

	got := String()
	want := "1.0.0 (commit: abc1234, built: 2026-01-01T00:00:00Z)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	// Reset to defaults
	Version = "dev"
	Commit = "unknown"
	Date = "unknown"
}
