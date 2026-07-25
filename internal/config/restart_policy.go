package config

import "fmt"

// RestartPolicy define cuándo un proceso hijo debe reiniciarse tras terminar.
type RestartPolicy string

const (
	RestartAlways    RestartPolicy = "always"
	RestartOnFailure RestartPolicy = "on-failure"
	RestartNever     RestartPolicy = "never"
)

// ParseRestartPolicy convierte un valor de configuración en una política válida.
func ParseRestartPolicy(raw string) (RestartPolicy, error) {
	if raw == "" {
		return RestartNever, nil
	}

	policy := RestartPolicy(raw)
	switch policy {
	case RestartAlways, RestartOnFailure, RestartNever:
		return policy, nil
	default:
		return "", fmt.Errorf("política de reinicio %q inválida: use always, on-failure o never", raw)
	}
}
