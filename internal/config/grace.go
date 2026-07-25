package config

import "time"

const defaultGracePeriodSeconds = 10.0

// GracePeriod devuelve la duración configurada para apagado ordenado de hijos.
func (c *Config) GracePeriod() time.Duration {
	if c.GracePeriodSeconds <= 0 {
		return time.Duration(defaultGracePeriodSeconds * float64(time.Second))
	}
	return time.Duration(c.GracePeriodSeconds * float64(time.Second))
}
