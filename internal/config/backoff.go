package config

import (
	"fmt"
	"math"
	"time"
)

const (
	defaultInitialSeconds = 1.0
	defaultFactor         = 2.0
	defaultMaxSeconds     = 30.0
)

// BackoffConfig define los parámetros de backoff exponencial para reinicios.
type BackoffConfig struct {
	InitialSeconds float64 `yaml:"initial_seconds"`
	Factor         float64 `yaml:"factor"`
	MaxSeconds     float64 `yaml:"max_seconds"`
}

// WithDefaults retorna una copia con valores por defecto para campos cero/negativos.
func (b BackoffConfig) WithDefaults() BackoffConfig {
	out := b
	if out.InitialSeconds <= 0 {
		out.InitialSeconds = defaultInitialSeconds
	}
	if out.Factor <= 0 {
		out.Factor = defaultFactor
	}
	if out.MaxSeconds <= 0 {
		out.MaxSeconds = defaultMaxSeconds
	}
	return out
}

// Validate comprueba que los parámetros de backoff sean consistentes.
func (b BackoffConfig) Validate() error {
	if b.InitialSeconds <= 0 {
		return fmt.Errorf("initial_seconds debe ser > 0, obtuvo %v", b.InitialSeconds)
	}
	if b.Factor <= 0 {
		return fmt.Errorf("factor debe ser > 0, obtuvo %v", b.Factor)
	}
	if b.MaxSeconds <= 0 {
		return fmt.Errorf("max_seconds debe ser > 0, obtuvo %v", b.MaxSeconds)
	}
	if b.InitialSeconds > b.MaxSeconds {
		return fmt.Errorf("initial_seconds (%v) no puede ser mayor que max_seconds (%v)",
			b.InitialSeconds, b.MaxSeconds)
	}
	return nil
}

// Merge retorna b con los campos en cero reemplazados por los valores de other.
func (b BackoffConfig) Merge(other BackoffConfig) BackoffConfig {
	out := b
	if out.InitialSeconds <= 0 {
		out.InitialSeconds = other.InitialSeconds
	}
	if out.Factor <= 0 {
		out.Factor = other.Factor
	}
	if out.MaxSeconds <= 0 {
		out.MaxSeconds = other.MaxSeconds
	}
	return out
}

// Duration calcula el tiempo de espera para un intento dado (0-indexed).
// Fórmula: min(initialSeconds * factor^attempt, maxSeconds).
func (b BackoffConfig) Duration(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	secs := b.InitialSeconds * math.Pow(b.Factor, float64(attempt))
	secs = math.Min(secs, b.MaxSeconds)
	return time.Duration(secs * float64(time.Second))
}
