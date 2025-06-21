package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateURL validates a URL and prevents SSRF attacks
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	// Only allow HTTP and HTTPS schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS schemes are allowed, got: %s", parsed.Scheme)
	}
	
	// Validate hostname
	if parsed.Hostname() == "" {
		return fmt.Errorf("URL must have a hostname")
	}
	
	// Check if hostname resolves to a private IP
	if err := validateHostnameNotPrivate(parsed.Hostname()); err != nil {
		return fmt.Errorf("hostname validation failed: %w", err)
	}
	
	return nil
}

// ValidateSSLTarget validates a target for SSL checks (URL or host:port format)
func ValidateSSLTarget(target string) error {
	if target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	
	// Try to parse as URL first
	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("invalid URL format: %w", err)
		}
		
		// For SSL checks, allow HTTP and HTTPS schemes
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("only HTTP and HTTPS schemes are allowed for SSL checks, got: %s", parsed.Scheme)
		}
		
		// Validate hostname
		if parsed.Hostname() == "" {
			return fmt.Errorf("URL must have a hostname")
		}
		
		return validateHostnameNotPrivate(parsed.Hostname())
	}
	
	// Try to parse as host:port format
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		// If no port specified, treat as hostname
		host = target
	}
	
	if host == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	
	return validateHostnameNotPrivate(host)
}

// validateHostnameNotPrivate checks if a hostname resolves to a private IP address
func validateHostnameNotPrivate(hostname string) error {
	// Skip validation for localhost in development
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return nil
	}
	
	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname: %w", err)
	}
	
	// Check if any resolved IP is private
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("hostname resolves to private IP address: %s", ip.String())
		}
	}
	
	return nil
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	// Private IPv4 ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",    // Loopback
		"169.254.0.0/16", // Link-local
	}
	
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	
	// Private IPv6 ranges
	if ip.To4() == nil { // IPv6
		// Check for IPv6 loopback
		if ip.IsLoopback() {
			return true
		}
		// Check for IPv6 link-local
		if ip.IsLinkLocalUnicast() {
			return true
		}
		// Check for IPv6 unique local addresses (fc00::/7)
		if len(ip) >= 1 && (ip[0]&0xfe) == 0xfc {
			return true
		}
	}
	
	return false
}

// ValidateHTTPHeaders validates HTTP headers to prevent injection attacks
func ValidateHTTPHeaders(headers map[string]string) error {
	if headers == nil {
		return nil
	}
	
	for key, value := range headers {
		// Check for newline characters that could enable header injection
		if strings.ContainsAny(key, "\r\n") {
			return fmt.Errorf("header name contains invalid characters: %s", key)
		}
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("header value contains invalid characters for key: %s", key)
		}
		
		// Check for empty header names
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("header name cannot be empty")
		}
		
		// Validate common security-sensitive headers
		keyLower := strings.ToLower(key)
		switch keyLower {
		case "host":
			// Validate host header format
			if err := validateHostHeader(value); err != nil {
				return fmt.Errorf("invalid host header: %w", err)
			}
		case "content-length":
			return fmt.Errorf("content-length header should not be set manually")
		}
	}
	
	return nil
}

// validateHostHeader validates the format of a Host header
func validateHostHeader(host string) error {
	if host == "" {
		return fmt.Errorf("host header cannot be empty")
	}
	
	// Parse as URL to validate format
	testURL := "http://" + host
	_, err := url.Parse(testURL)
	if err != nil {
		return fmt.Errorf("invalid host format: %w", err)
	}
	
	return nil
}

// ValidateFilePath validates file paths to prevent directory traversal
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	
	// Check for directory traversal patterns
	if strings.Contains(path, "..") {
		return fmt.Errorf("file path contains directory traversal pattern: %s", path)
	}
	
	// Check for absolute paths to sensitive locations
	sensitivePaths := []string{
		"/etc/",
		"/proc/",
		"/sys/",
		"/dev/",
		"/root/",
		"/home/",
		"/usr/",
	}
	
	for _, sensitive := range sensitivePaths {
		if strings.HasPrefix(path, sensitive) {
			return fmt.Errorf("access to sensitive path not allowed: %s", path)
		}
	}
	
	return nil
}