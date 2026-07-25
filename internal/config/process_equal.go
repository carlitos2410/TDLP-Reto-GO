package config

import "reflect"

// ProcessConfigEqual indica si dos configuraciones de proceso son equivalentes.
func ProcessConfigEqual(a, b ProcessConfig) bool {
	return reflect.DeepEqual(a, b)
}
