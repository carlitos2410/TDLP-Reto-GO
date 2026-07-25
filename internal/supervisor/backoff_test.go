package supervisor

import (
	"testing"
	"time"

	"supervisor-procesos/internal/config"
)

func TestBackoffNext(t *testing.T) {
	cfg := config.BackoffConfig{
		InitialSeconds: 1,
		Factor:         2,
		MaxSeconds:     30,
	}
	b := NewBackoff(cfg)

	tests := []struct {
		name       string
		attempts   int
		wantApprox time.Duration
	}{
		{"first", 1, 1 * time.Second},
		{"second", 1, 2 * time.Second},
		{"third", 1, 4 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.attempts; i++ {
				got := b.Next("worker")
				if i == tt.attempts-1 {
					if got != tt.wantApprox {
						t.Errorf("Next() = %v, want %v", got, tt.wantApprox)
					}
				}
			}
		})
	}
}

func TestBackoffMaxCap(t *testing.T) {
	cfg := config.BackoffConfig{
		InitialSeconds: 1,
		Factor:         2,
		MaxSeconds:     5,
	}
	b := NewBackoff(cfg)

	for i := 0; i < 20; i++ {
		got := b.Next("worker")
		if got > 5*time.Second {
			t.Fatalf("Next() = %v, should never exceed max_seconds (5s)", got)
		}
	}
}

func TestBackoffReset(t *testing.T) {
	cfg := config.BackoffConfig{
		InitialSeconds: 1,
		Factor:         2,
		MaxSeconds:     30,
	}
	b := NewBackoff(cfg)

	b.Next("worker")
	b.Next("worker")
	b.Next("worker")

	b.Reset("worker")

	got := b.Next("worker")
	if got != 1*time.Second {
		t.Fatalf("después de Reset(), Next() = %v, want 1s", got)
	}
}

func TestBackoffIndependentProcesses(t *testing.T) {
	cfg := config.BackoffConfig{
		InitialSeconds: 1,
		Factor:         2,
		MaxSeconds:     30,
	}
	b := NewBackoff(cfg)

	b.Next("worker-a")
	b.Next("worker-a")

	gotB := b.Next("worker-b")
	if gotB != 1*time.Second {
		t.Fatalf("Next() para worker-b = %v, want 1s (debería ser independiente)", gotB)
	}
}

func TestShouldStopFalse(t *testing.T) {
	cfg := config.BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}
	b := NewBackoff(cfg)

	procCfg := config.ProcessConfig{Name: "worker", MaxRetries: 5}
	if b.ShouldStop(procCfg) {
		t.Fatal("ShouldStop() = true con 0 intentos, want false")
	}
}

func TestShouldStopTrue(t *testing.T) {
	cfg := config.BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}
	b := NewBackoff(cfg)

	procCfg := config.ProcessConfig{Name: "worker", MaxRetries: 3}
	b.Next("worker")
	b.Next("worker")
	b.Next("worker")

	if !b.ShouldStop(procCfg) {
		t.Fatal("ShouldStop() = false con 3/3 intentos, want true")
	}
}

func TestShouldStopUnlimitedRetries(t *testing.T) {
	cfg := config.BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}
	b := NewBackoff(cfg)

	procCfg := config.ProcessConfig{Name: "worker", MaxRetries: 0}
	for i := 0; i < 100; i++ {
		b.Next("worker")
	}

	if b.ShouldStop(procCfg) {
		t.Fatal("ShouldStop() = true con MaxRetries=0 (ilimitado), want false")
	}
}

func TestCurrentAttempt(t *testing.T) {
	cfg := config.BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}
	b := NewBackoff(cfg)

	if got := b.CurrentAttempt("worker"); got != 0 {
		t.Fatalf("CurrentAttempt() = %d, want 0", got)
	}

	b.Next("worker")
	b.Next("worker")

	if got := b.CurrentAttempt("worker"); got != 2 {
		t.Fatalf("CurrentAttempt() = %d, want 2", got)
	}
}
