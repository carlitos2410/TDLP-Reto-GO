package config

import (
	"testing"
	"time"
)

func TestGracePeriodDefault(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	if got := cfg.GracePeriod(); got != 10*time.Second {
		t.Fatalf("GracePeriod() = %s, want 10s", got)
	}
}

func TestGracePeriodCustom(t *testing.T) {
	t.Parallel()

	cfg := &Config{GracePeriodSeconds: 5}
	if got := cfg.GracePeriod(); got != 5*time.Second {
		t.Fatalf("GracePeriod() = %s, want 5s", got)
	}
}

func TestProcessConfigEqualDetectsChanges(t *testing.T) {
	t.Parallel()

	a := ProcessConfig{Name: "a", Command: "echo"}
	b := ProcessConfig{Name: "a", Command: "echo", Args: []string{"x"}}
	if ProcessConfigEqual(a, b) {
		t.Fatal("ProcessConfigEqual() debería detectar args distintos")
	}
}
