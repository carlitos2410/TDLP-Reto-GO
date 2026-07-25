package config

import (
	"math"
	"testing"
)

func TestWithDefaults(t *testing.T) {
	b := BackoffConfig{}
	got := b.WithDefaults()

	if got.InitialSeconds != defaultInitialSeconds {
		t.Fatalf("WithDefaults().InitialSeconds = %v, want %v", got.InitialSeconds, defaultInitialSeconds)
	}
	if got.Factor != defaultFactor {
		t.Fatalf("WithDefaults().Factor = %v, want %v", got.Factor, defaultFactor)
	}
	if got.MaxSeconds != defaultMaxSeconds {
		t.Fatalf("WithDefaults().MaxSeconds = %v, want %v", got.MaxSeconds, defaultMaxSeconds)
	}
}

func TestWithDefaultsPreservesExplicit(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 5, Factor: 3, MaxSeconds: 60}
	got := b.WithDefaults()

	if got.InitialSeconds != 5 {
		t.Fatalf("WithDefaults().InitialSeconds = %v, want 5", got.InitialSeconds)
	}
	if got.Factor != 3 {
		t.Fatalf("WithDefaults().Factor = %v, want 3", got.Factor)
	}
	if got.MaxSeconds != 60 {
		t.Fatalf("WithDefaults().MaxSeconds = %v, want 60", got.MaxSeconds)
	}
}

func TestValidateAcceptsValid(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}
	if err := b.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsZeroInitial(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 0, Factor: 2, MaxSeconds: 30}
	if err := b.Validate(); err == nil {
		t.Fatal("Validate() debería fallar con initial_seconds = 0")
	}
}

func TestValidateRejectsZeroFactor(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 1, Factor: 0, MaxSeconds: 30}
	if err := b.Validate(); err == nil {
		t.Fatal("Validate() debería fallar con factor = 0")
	}
}

func TestValidateRejectsZeroMax(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 0}
	if err := b.Validate(); err == nil {
		t.Fatal("Validate() debería fallar con max_seconds = 0")
	}
}

func TestValidateRejectsInitialGreaterThanMax(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 60, Factor: 2, MaxSeconds: 30}
	if err := b.Validate(); err == nil {
		t.Fatal("Validate() debería fallar cuando initial_seconds > max_seconds")
	}
}

func TestValidateRejectsNegativeFactor(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 1, Factor: -1, MaxSeconds: 30}
	if err := b.Validate(); err == nil {
		t.Fatal("Validate() debería fallar con factor negativo")
	}
}

func TestMergeOverridesZeros(t *testing.T) {
	local := BackoffConfig{}
	global := BackoffConfig{InitialSeconds: 2, Factor: 3, MaxSeconds: 45}
	got := local.Merge(global)

	if got.InitialSeconds != 2 {
		t.Fatalf("Merge().InitialSeconds = %v, want 2", got.InitialSeconds)
	}
	if got.Factor != 3 {
		t.Fatalf("Merge().Factor = %v, want 3", got.Factor)
	}
	if got.MaxSeconds != 45 {
		t.Fatalf("Merge().MaxSeconds = %v, want 45", got.MaxSeconds)
	}
}

func TestMergeKeepsLocalValues(t *testing.T) {
	local := BackoffConfig{InitialSeconds: 5, Factor: 4, MaxSeconds: 100}
	global := BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}
	got := local.Merge(global)

	if got.InitialSeconds != 5 {
		t.Fatalf("Merge().InitialSeconds = %v, want 5", got.InitialSeconds)
	}
	if got.Factor != 4 {
		t.Fatalf("Merge().Factor = %v, want 4", got.Factor)
	}
	if got.MaxSeconds != 100 {
		t.Fatalf("Merge().MaxSeconds = %v, want 100", got.MaxSeconds)
	}
}

func TestMergePartialOverride(t *testing.T) {
	local := BackoffConfig{InitialSeconds: 5}
	global := BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}
	got := local.Merge(global)

	if got.InitialSeconds != 5 {
		t.Fatalf("Merge().InitialSeconds = %v, want 5", got.InitialSeconds)
	}
	if got.Factor != 2 {
		t.Fatalf("Merge().Factor = %v, want 2", got.Factor)
	}
	if got.MaxSeconds != 30 {
		t.Fatalf("Merge().MaxSeconds = %v, want 30", got.MaxSeconds)
	}
}

func TestBackoffDuration(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 30}

	tests := []struct {
		attempt    int
		wantApprox float64
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 8},
		{5, 16},
		{6, 30},
		{10, 30},
	}

	for _, tt := range tests {
		got := b.Duration(tt.attempt)
		if math.Abs(got.Seconds()-tt.wantApprox) > 0.01 {
			t.Errorf("Duration(%d) = %v, want ~%vs", tt.attempt, got, tt.wantApprox)
		}
	}
}
