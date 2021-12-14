package utils

import (
	"os"
)

// EnvVarOrDefault returns the value of the environment variable named "envName" if found or "defaultVal" otherwise
func EnvVarOrDefault(envName, defaultVal string) string {
	if val := os.Getenv(envName); val == "" {
		return defaultVal
	} else {
		return val
	}
}
