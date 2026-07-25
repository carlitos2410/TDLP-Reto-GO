package config

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
