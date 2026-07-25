package config

const defaultAPIListen = ":8080"

// APIListenAddress devuelve la dirección HTTP del API de control.
func (c *Config) APIListenAddress() string {
	if c.APIListen == "" {
		return defaultAPIListen
	}
	return c.APIListen
}
