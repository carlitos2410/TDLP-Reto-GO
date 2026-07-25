package config

import "testing"

func TestAPIListenAddressDefault(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	if got := cfg.APIListenAddress(); got != ":8080" {
		t.Fatalf("APIListenAddress() = %q, want :8080", got)
	}
}
