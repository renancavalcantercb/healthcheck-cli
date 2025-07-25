package security

import (
	"strings"
)

// MaskEmail masks sensitive parts of an email address
// Example: user@example.com -> us***@example.com
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	
	localPart := parts[0]
	domain := parts[1]
	
	if len(localPart) <= 2 {
		return "***@" + domain
	}
	
	return localPart[:2] + "***@" + domain
}

// MaskEmailList masks a list of email addresses
func MaskEmailList(emails []string) []string {
	masked := make([]string, len(emails))
	for i, email := range emails {
		masked[i] = MaskEmail(email)
	}
	return masked
}

// MaskURL masks sensitive parts of a URL (query parameters and path)
// Example: https://api.service.com/webhook/secret123 -> https://api.service.com/webhook/***
func MaskURL(url string) string {
	if url == "" {
		return ""
	}
	
	// Find the last slash and mask everything after it
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash == -1 || lastSlash >= len(url)-1 {
		return url
	}
	
	// Keep protocol and domain, mask the path
	protocolEnd := strings.Index(url, "://")
	if protocolEnd == -1 {
		return "***"
	}
	
	domainEnd := strings.Index(url[protocolEnd+3:], "/")
	if domainEnd == -1 {
		return url
	}
	
	baseURL := url[:protocolEnd+3+domainEnd+1]
	return baseURL + "***"
}

// SanitizeForLogs removes or masks sensitive information from log messages
func SanitizeForLogs(message string) string {
	// Remove common sensitive patterns
	patterns := []string{
		"password=",
		"token=",
		"key=",
		"secret=",
		"api_key=",
		"webhook=",
	}
	
	result := message
	lowerResult := strings.ToLower(result)
	
	for _, pattern := range patterns {
		if strings.Contains(lowerResult, pattern) {
			// Find the pattern and replace the value after it
			start := strings.Index(lowerResult, pattern)
			if start >= 0 {
				// Find the end of the value (space or end of string)
				valueStart := start + len(pattern)
				valueEnd := strings.Index(lowerResult[valueStart:], " ")
				if valueEnd == -1 {
					valueEnd = len(lowerResult)
				} else {
					valueEnd += valueStart
				}
				
				// Replace the original case pattern with masked version
				originalPattern := result[start:start+len(pattern)]
				result = result[:start] + strings.ToLower(originalPattern) + "***" + result[valueEnd:]
				lowerResult = strings.ToLower(result)
			}
		}
	}
	
	return result
}