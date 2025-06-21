package env

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ExpandEnvironmentVariables expands environment variables in a string
// Supports formats: ${VAR_NAME}, ${VAR_NAME:default_value}, $VAR_NAME
func ExpandEnvironmentVariables(input string) string {
	// Pattern for ${VAR_NAME} and ${VAR_NAME:default}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)
	
	result := re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name and default value
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		
		varName := parts[1]
		defaultValue := ""
		if len(parts) > 2 {
			defaultValue = parts[2]
		}
		
		// Get environment variable value
		if value := os.Getenv(varName); value != "" {
			return value
		}
		
		// Return default value if provided, otherwise keep original
		if defaultValue != "" {
			return defaultValue
		}
		
		// If no default and env var is empty, keep the placeholder
		return match
	})
	
	// Pattern for $VAR_NAME (without braces)
	simpleRe := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	result = simpleRe.ReplaceAllStringFunc(result, func(match string) string {
		varName := strings.TrimPrefix(match, "$")
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match
	})
	
	return result
}

// ExpandEnvironmentVariablesInMap recursively expands environment variables in a map
func ExpandEnvironmentVariablesInMap(data map[string]interface{}) {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			data[key] = ExpandEnvironmentVariables(v)
		case map[string]interface{}:
			ExpandEnvironmentVariablesInMap(v)
		case []interface{}:
			expandEnvironmentVariablesInSlice(v)
		}
	}
}

// expandEnvironmentVariablesInSlice recursively expands environment variables in a slice
func expandEnvironmentVariablesInSlice(slice []interface{}) {
	for i, value := range slice {
		switch v := value.(type) {
		case string:
			slice[i] = ExpandEnvironmentVariables(v)
		case map[string]interface{}:
			ExpandEnvironmentVariablesInMap(v)
		case []interface{}:
			expandEnvironmentVariablesInSlice(v)
		}
	}
}

// ValidateRequiredEnvVars checks if required environment variables are set
func ValidateRequiredEnvVars(vars []string) error {
	var missing []string
	
	for _, varName := range vars {
		if os.Getenv(varName) == "" {
			missing = append(missing, varName)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("required environment variables not set: %s", strings.Join(missing, ", "))
	}
	
	return nil
}

// GetEnvWithDefault returns the value of an environment variable or a default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetEnvIfNotExists sets an environment variable only if it's not already set
func SetEnvIfNotExists(key, value string) error {
	if os.Getenv(key) == "" {
		return os.Setenv(key, value)
	}
	return nil
}

// LoadEnvFile loads environment variables from a .env file
func LoadEnvFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read env file %s: %w", filename, err)
	}
	
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format in env file %s at line %d: %s", filename, i+1, line)
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			value = value[1 : len(value)-1]
		}
		
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}
	
	return nil
}