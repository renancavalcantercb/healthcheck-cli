package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnvironmentVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("EMAIL_PASSWORD", "secret123")
	os.Setenv("API_TOKEN", "token456")
	defer func() {
		os.Unsetenv("TEST_VAR")
		os.Unsetenv("EMAIL_PASSWORD")
		os.Unsetenv("API_TOKEN")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SimpleVariable",
			input:    "${TEST_VAR}",
			expected: "test_value",
		},
		{
			name:     "VariableWithDefault_VarExists",
			input:    "${TEST_VAR:default}",
			expected: "test_value",
		},
		{
			name:     "VariableWithDefault_VarMissing",
			input:    "${MISSING_VAR:default_value}",
			expected: "default_value",
		},
		{
			name:     "VariableWithoutBraces",
			input:    "$TEST_VAR",
			expected: "test_value",
		},
		{
			name:     "MixedText",
			input:    "prefix_${TEST_VAR}_suffix",
			expected: "prefix_test_value_suffix",
		},
		{
			name:     "MultipleVariables",
			input:    "${EMAIL_PASSWORD} and ${API_TOKEN}",
			expected: "secret123 and token456",
		},
		{
			name:     "NoVariables",
			input:    "just plain text",
			expected: "just plain text",
		},
		{
			name:     "EmptyVariable",
			input:    "${EMPTY_VAR}",
			expected: "${EMPTY_VAR}",
		},
		{
			name:     "EmptyVariableWithDefault",
			input:    "${EMPTY_VAR:fallback}",
			expected: "fallback",
		},
		{
			name:     "ComplexConfiguration",
			input:    "smtp://${EMAIL_PASSWORD}@smtp.gmail.com:587",
			expected: "smtp://secret123@smtp.gmail.com:587",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvironmentVariables(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandEnvironmentVariablesInMap(t *testing.T) {
	// Set up test environment variables
	os.Setenv("DB_PASSWORD", "dbsecret")
	os.Setenv("API_KEY", "apikey123")
	defer func() {
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("API_KEY")
	}()

	testMap := map[string]interface{}{
		"database": map[string]interface{}{
			"host":     "localhost",
			"password": "${DB_PASSWORD}",
			"port":     5432,
		},
		"api": map[string]interface{}{
			"key":     "${API_KEY}",
			"timeout": 30,
		},
		"servers": []interface{}{
			"server1",
			"${API_KEY}_server2",
			map[string]interface{}{
				"name": "server3",
				"auth": "${DB_PASSWORD}",
			},
		},
		"simple": "no_variables_here",
	}

	ExpandEnvironmentVariablesInMap(testMap)

	// Check database config
	dbConfig := testMap["database"].(map[string]interface{})
	assert.Equal(t, "localhost", dbConfig["host"])
	assert.Equal(t, "dbsecret", dbConfig["password"])
	assert.Equal(t, 5432, dbConfig["port"])

	// Check API config
	apiConfig := testMap["api"].(map[string]interface{})
	assert.Equal(t, "apikey123", apiConfig["key"])
	assert.Equal(t, 30, apiConfig["timeout"])

	// Check servers array
	servers := testMap["servers"].([]interface{})
	assert.Equal(t, "server1", servers[0])
	assert.Equal(t, "apikey123_server2", servers[1])
	
	server3 := servers[2].(map[string]interface{})
	assert.Equal(t, "server3", server3["name"])
	assert.Equal(t, "dbsecret", server3["auth"])

	// Check simple string
	assert.Equal(t, "no_variables_here", testMap["simple"])
}

func TestValidateRequiredEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("REQUIRED_VAR1", "value1")
	os.Setenv("REQUIRED_VAR2", "value2")
	defer func() {
		os.Unsetenv("REQUIRED_VAR1")
		os.Unsetenv("REQUIRED_VAR2")
	}()

	tests := []struct {
		name     string
		vars     []string
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "AllVariablesPresent",
			vars:    []string{"REQUIRED_VAR1", "REQUIRED_VAR2"},
			wantErr: false,
		},
		{
			name:    "SomeVariablesMissing",
			vars:    []string{"REQUIRED_VAR1", "MISSING_VAR"},
			wantErr: true,
			errMsg:  "required environment variables not set: MISSING_VAR",
		},
		{
			name:    "AllVariablesMissing",
			vars:    []string{"MISSING_VAR1", "MISSING_VAR2"},
			wantErr: true,
			errMsg:  "required environment variables not set: MISSING_VAR1, MISSING_VAR2",
		},
		{
			name:    "EmptyList",
			vars:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequiredEnvVars(tt.vars)
			
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	os.Setenv("EXISTING_VAR", "existing_value")
	defer os.Unsetenv("EXISTING_VAR")

	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "ExistingVariable",
			key:          "EXISTING_VAR",
			defaultValue: "default",
			expected:     "existing_value",
		},
		{
			name:         "MissingVariable",
			key:          "MISSING_VAR",
			defaultValue: "default_value",
			expected:     "default_value",
		},
		{
			name:         "EmptyDefault",
			key:          "MISSING_VAR",
			defaultValue: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEnvWithDefault(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetEnvIfNotExists(t *testing.T) {
	// Clean up any existing variable
	os.Unsetenv("TEST_SET_VAR")
	defer os.Unsetenv("TEST_SET_VAR")

	// Test setting when variable doesn't exist
	err := SetEnvIfNotExists("TEST_SET_VAR", "new_value")
	assert.NoError(t, err)
	assert.Equal(t, "new_value", os.Getenv("TEST_SET_VAR"))

	// Test not overwriting when variable exists
	err = SetEnvIfNotExists("TEST_SET_VAR", "different_value")
	assert.NoError(t, err)
	assert.Equal(t, "new_value", os.Getenv("TEST_SET_VAR")) // Should remain unchanged
}

func TestLoadEnvFile(t *testing.T) {
	// Create a temporary .env file
	envContent := `# This is a comment
FIRST_VAR=first_value
SECOND_VAR="quoted_value"
THIRD_VAR='single_quoted'
FOURTH_VAR=value with spaces

# Another comment
FIFTH_VAR=
SIXTH_VAR=no_quotes`

	tempFile := "/tmp/test.env"
	err := os.WriteFile(tempFile, []byte(envContent), 0644)
	require.NoError(t, err)
	defer os.Remove(tempFile)

	// Clean up environment variables
	vars := []string{"FIRST_VAR", "SECOND_VAR", "THIRD_VAR", "FOURTH_VAR", "FIFTH_VAR", "SIXTH_VAR"}
	for _, v := range vars {
		os.Unsetenv(v)
	}
	defer func() {
		for _, v := range vars {
			os.Unsetenv(v)
		}
	}()

	// Load the .env file
	err = LoadEnvFile(tempFile)
	assert.NoError(t, err)

	// Verify variables were set correctly
	assert.Equal(t, "first_value", os.Getenv("FIRST_VAR"))
	assert.Equal(t, "quoted_value", os.Getenv("SECOND_VAR"))
	assert.Equal(t, "single_quoted", os.Getenv("THIRD_VAR"))
	assert.Equal(t, "value with spaces", os.Getenv("FOURTH_VAR"))
	assert.Equal(t, "", os.Getenv("FIFTH_VAR"))
	assert.Equal(t, "no_quotes", os.Getenv("SIXTH_VAR"))
}

func TestLoadEnvFile_Errors(t *testing.T) {
	// Test non-existent file
	err := LoadEnvFile("/tmp/nonexistent.env")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read env file")

	// Test invalid format
	invalidContent := `INVALID_LINE_WITHOUT_EQUALS`
	tempFile := "/tmp/invalid.env"
	err = os.WriteFile(tempFile, []byte(invalidContent), 0644)
	require.NoError(t, err)
	defer os.Remove(tempFile)

	err = LoadEnvFile(tempFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestSecurityPatterns(t *testing.T) {
	// Test that sensitive patterns are properly expanded
	os.Setenv("EMAIL_PASSWORD", "secret_password")
	os.Setenv("API_TOKEN", "bearer_token_123")
	os.Setenv("WEBHOOK_URL", "https://hooks.slack.com/services/secret")
	defer func() {
		os.Unsetenv("EMAIL_PASSWORD")
		os.Unsetenv("API_TOKEN")
		os.Unsetenv("WEBHOOK_URL")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "EmailPassword",
			input:    "password: ${EMAIL_PASSWORD}",
			expected: "password: secret_password",
		},
		{
			name:     "APIToken",
			input:    "Authorization: Bearer ${API_TOKEN}",
			expected: "Authorization: Bearer bearer_token_123",
		},
		{
			name:     "WebhookURL",
			input:    "webhook_url: ${WEBHOOK_URL}",
			expected: "webhook_url: https://hooks.slack.com/services/secret",
		},
		{
			name:     "MultipleSecrets",
			input:    "user: admin, password: ${EMAIL_PASSWORD}, token: ${API_TOKEN}",
			expected: "user: admin, password: secret_password, token: bearer_token_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvironmentVariables(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}