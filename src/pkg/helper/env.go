package helper

import (
	"fmt"
	"os"
)

// GetEnvWithDefault retrieves the value of the environment variable named by envName.
// If the environment variable is not set or is empty, it returns defaultValue.
func GetEnvWithDefault(envName string, defaultValue string) string {
	if v := os.Getenv(envName); v != "" {
		return v
	}
	return defaultValue
}

// GetEnvWithError retrieves the value of the environment variable named by envName.
// It returns an error if the environment variable is not set.
func GetEnvWithError(envName string) (string, error) {
	value, exists := os.LookupEnv(envName)
	if !exists {
		return "", fmt.Errorf("environment variable %q is not set", envName)
	}
	return value, nil
}
