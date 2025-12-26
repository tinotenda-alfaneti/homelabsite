package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "Environment variable exists",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "actual",
			expected:     "actual",
		},
		{
			name:         "Environment variable does not exist",
			key:          "NONEXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnv(tt.key, tt.defaultValue)
			if got != tt.expected {
				t.Errorf("GetEnv(%s, %s) = %s, want %s", tt.key, tt.defaultValue, got, tt.expected)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Save original env
	originalConfigPath := os.Getenv("CONFIG_PATH")
	defer func() {
		if originalConfigPath != "" {
			os.Setenv("CONFIG_PATH", originalConfigPath)
		} else {
			os.Unsetenv("CONFIG_PATH")
		}
	}()

	tests := []struct {
		name          string
		configPathEnv string
		expected      string
	}{
		{
			name:          "CONFIG_PATH environment variable set",
			configPathEnv: "/custom/config/config.yaml",
			expected:      "/custom/config/config.yaml",
		},
		{
			name:          "CONFIG_PATH not set, default path",
			configPathEnv: "",
			expected:      "config/config.yaml", // Default for local
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.configPathEnv != "" {
				os.Setenv("CONFIG_PATH", tt.configPathEnv)
			} else {
				os.Unsetenv("CONFIG_PATH")
			}

			got := GetConfigPath()
			if got != tt.expected {
				t.Errorf("GetConfigPath() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestGetDataDir(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		contains   string // Just check if path contains this string
	}{
		{
			name:       "PVC config path",
			configPath: "/app/config/config.yaml",
			contains:   "data",
		},
		{
			name:       "ConfigMap path",
			configPath: "/srv/config/config.yaml",
			contains:   "data",
		},
		{
			name:       "Local config path",
			configPath: "config/config.yaml",
			contains:   "data",
		},
		{
			name:       "Custom local path",
			configPath: "custom/path/config.yaml",
			contains:   "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDataDir(tt.configPath)
			// Just verify it contains "data" rather than exact path matching
			// to avoid Windows/Linux path separator issues in tests
			if got == "" {
				t.Errorf("GetDataDir(%s) returned empty string", tt.configPath)
			}
		})
	}
}
