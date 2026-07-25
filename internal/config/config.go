package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config representa la configuración global del supervisor.
type Config struct {
	LogDir             string          `yaml:"log_dir"`
	GracePeriodSeconds float64         `yaml:"grace_period_seconds"`
	BackoffDefaults    BackoffConfig   `yaml:"backoff_defaults"`
	MaxRetriesDefault  int             `yaml:"max_retries_default"`
	APIListen          string          `yaml:"api_listen"`
	Processes          []ProcessConfig `yaml:"processes"`
}

// ProcessConfig describe un proceso hijo supervisado.
type ProcessConfig struct {
	Name          string            `yaml:"name"`
	Command       string            `yaml:"command"`
	Args          []string          `yaml:"args"`
	Env           map[string]string `yaml:"env"`
	WorkDir       string            `yaml:"work_dir"`
	StdoutLog     string            `yaml:"stdout_log"`
	StderrLog     string            `yaml:"stderr_log"`
	RestartPolicy RestartPolicy     `yaml:"restart_policy"`
	Backoff       BackoffConfig     `yaml:"backoff"`
	MaxRetries    int               `yaml:"max_retries"`
}

// Load lee y valida un archivo de configuración YAML.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leer configuración %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsear YAML %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate comprueba que la configuración sea coherente antes de arrancar procesos.
func (c *Config) Validate() error {
	if c.LogDir == "" {
		c.LogDir = "logs"
	}

	c.BackoffDefaults = c.BackoffDefaults.WithDefaults()

	if err := c.BackoffDefaults.Validate(); err != nil {
		return fmt.Errorf("backoff_defaults: %w", err)
	}

	if len(c.Processes) == 0 {
		return fmt.Errorf("la configuración debe definir al menos un proceso")
	}

	seen := make(map[string]struct{}, len(c.Processes))
	for i := range c.Processes {
		p := &c.Processes[i]
		if p.Name == "" {
			return fmt.Errorf("proceso #%d: el campo name es obligatorio", i+1)
		}
		if _, dup := seen[p.Name]; dup {
			return fmt.Errorf("proceso %q: nombre duplicado", p.Name)
		}
		seen[p.Name] = struct{}{}

		if p.Command == "" {
			return fmt.Errorf("proceso %q: el campo command es obligatorio", p.Name)
		}

		if p.WorkDir != "" {
			info, err := os.Stat(p.WorkDir)
			if err != nil {
				return fmt.Errorf("proceso %q: work_dir %q inválido: %w", p.Name, p.WorkDir, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("proceso %q: work_dir %q no es un directorio", p.Name, p.WorkDir)
			}
		}

		if p.StdoutLog == "" {
			p.StdoutLog = filepath.Join(c.LogDir, p.Name+".stdout.log")
		}
		if p.StderrLog == "" {
			p.StderrLog = filepath.Join(c.LogDir, p.Name+".stderr.log")
		}

		policy, err := ParseRestartPolicy(string(p.RestartPolicy))
		if err != nil {
			return fmt.Errorf("proceso %q: %w", p.Name, err)
		}
		p.RestartPolicy = policy

		p.Backoff = p.Backoff.Merge(c.BackoffDefaults)
		if err := p.Backoff.Validate(); err != nil {
			return fmt.Errorf("proceso %q: backoff: %w", p.Name, err)
		}

		if p.MaxRetries == 0 {
			p.MaxRetries = c.MaxRetriesDefault
		}
	}

	return nil
}
