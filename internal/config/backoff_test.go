package config

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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

func TestBackoffConfigYAML(t *testing.T) {
	input := `
initial_seconds: 2.5
factor: 3
max_seconds: 60
`
	var b BackoffConfig
	if err := yaml.Unmarshal([]byte(input), &b); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if b.InitialSeconds != 2.5 {
		t.Fatalf("InitialSeconds = %v, want 2.5", b.InitialSeconds)
	}
	if b.Factor != 3 {
		t.Fatalf("Factor = %v, want 3", b.Factor)
	}
	if b.MaxSeconds != 60 {
		t.Fatalf("MaxSeconds = %v, want 60", b.MaxSeconds)
	}
}

func TestBackoffConfigYAMLFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "supervisor.yaml")
	content := `
log_dir: logs
backoff_defaults:
  initial_seconds: 1
  factor: 2
  max_seconds: 30
processes:
  - name: worker
    command: echo
    backoff:
      initial_seconds: 5
      factor: 3
      max_seconds: 60
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir config temporal: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got := cfg.Processes[0].Backoff
	if got.InitialSeconds != 5 {
		t.Fatalf("Backoff.InitialSeconds = %v, want 5", got.InitialSeconds)
	}
	if got.Factor != 3 {
		t.Fatalf("Backoff.Factor = %v, want 3", got.Factor)
	}
	if got.MaxSeconds != 60 {
		t.Fatalf("Backoff.MaxSeconds = %v, want 60", got.MaxSeconds)
	}
}

func TestBackoffConfigYAMLZeroDefaults(t *testing.T) {
	input := `initial_seconds: 0`
	var b BackoffConfig
	if err := yaml.Unmarshal([]byte(input), &b); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	got := b.WithDefaults()
	if got.InitialSeconds != defaultInitialSeconds {
		t.Fatalf("WithDefaults().InitialSeconds = %v, want %v", got.InitialSeconds, defaultInitialSeconds)
	}
}

func TestDurationNegativeAttemptClampsToZero(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 2, Factor: 2, MaxSeconds: 30}
	got := b.Duration(-5)
	if got != 2*time.Second {
		t.Fatalf("Duration(-5) = %v, want 2s (debería tratar como attempt 0)", got)
	}
}

func TestDurationFactorOne(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 5, Factor: 1, MaxSeconds: 30}
	for i := 0; i < 10; i++ {
		got := b.Duration(i)
		if got != 5*time.Second {
			t.Fatalf("Duration(%d) = %v con factor=1, want siempre 5s", i, got)
		}
	}
}

func TestDurationLargeAttemptCapsAtMax(t *testing.T) {
	b := BackoffConfig{InitialSeconds: 1, Factor: 2, MaxSeconds: 10}
	got := b.Duration(100)
	if got != 10*time.Second {
		t.Fatalf("Duration(100) = %v, want max=10s", got)
	}
}
