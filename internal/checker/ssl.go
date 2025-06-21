package checker

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/security"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// SSLChecker implements SSL certificate checks
type SSLChecker struct {
	timeout time.Duration
}

// NewSSLChecker creates a new SSL checker
func NewSSLChecker(timeout time.Duration) *SSLChecker {
	return &SSLChecker{
		timeout: timeout,
	}
}

// Name returns the checker name
func (s *SSLChecker) Name() string {
	return "SSL"
}

// Check performs an SSL certificate check
func (s *SSLChecker) Check(check types.CheckConfig) types.Result {
	start := time.Now()
	
	result := types.Result{
		Name:      check.Name,
		URL:       check.URL,
		Timestamp: start,
	}
	
	// Validate target for security (supports both URL and host:port formats)
	if err := security.ValidateSSLTarget(check.URL); err != nil {
		result.Status = types.StatusError
		result.Error = fmt.Sprintf("Target validation failed: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}
	
	// Parse the URL to extract host and port
	host, port, err := s.parseHostPort(check.URL)
	if err != nil {
		result.Status = types.StatusError
		result.Error = fmt.Sprintf("Failed to parse host:port from URL: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}
	
	// Connect to the server and get certificate info
	certInfo, err := s.getCertificateInfo(host, port)
	duration := time.Since(start)
	result.ResponseTime = duration
	
	if err != nil {
		result.Status = types.StatusDown
		result.Error = fmt.Sprintf("Failed to get certificate info: %v", err)
		return result
	}
	
	result.CertInfo = certInfo
	
	// Validate certificate
	if err := s.validateCertificate(certInfo, check.Expected); err != nil {
		result.Status = types.StatusWarning
		result.Error = fmt.Sprintf("Certificate validation failed: %v", err)
		return result
	}
	
	// Check response time performance
	if check.Expected.ResponseTimeMax > 0 && duration > check.Expected.ResponseTimeMax {
		result.Status = types.StatusSlow
		result.Error = fmt.Sprintf("SSL handshake time %v exceeds maximum %v", duration, check.Expected.ResponseTimeMax)
		return result
	}
	
	// All checks passed
	result.Status = types.StatusUp
	return result
}

// parseHostPort extracts host and port from URL
func (s *SSLChecker) parseHostPort(rawURL string) (string, string, error) {
	// If it looks like host:port format, use it directly
	if !strings.Contains(rawURL, "://") {
		host, port, err := net.SplitHostPort(rawURL)
		if err != nil {
			// If no port specified, assume 443 for SSL
			return rawURL, "443", nil
		}
		return host, port, nil
	}
	
	// Parse as URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL format: %w", err)
	}
	
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	
	// Default ports based on scheme
	if port == "" {
		switch parsedURL.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			port = "443" // Default to SSL port
		}
	}
	
	return host, port, nil
}

// getCertificateInfo connects to the server and retrieves certificate information
func (s *SSLChecker) getCertificateInfo(host, port string) (*types.CertInfo, error) {
	// Create connection with timeout
	dialer := &net.Dialer{
		Timeout: s.timeout,
	}
	
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false, // Always verify certificates
	})
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}
	defer conn.Close()
	
	// Get certificate chain
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found")
	}
	
	// Use the first certificate (leaf certificate)
	cert := certs[0]
	
	// Calculate days to expiry
	now := time.Now()
	daysToExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
	
	// Check if certificate is currently valid
	isValid := now.After(cert.NotBefore) && now.Before(cert.NotAfter)
	
	certInfo := &types.CertInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		ExpiryDate:   cert.NotAfter,
		DaysToExpiry: daysToExpiry,
		IsValid:      isValid,
		CommonName:   cert.Subject.CommonName,
		DNSNames:     cert.DNSNames,
	}
	
	return certInfo, nil
}

// validateCertificate validates the certificate against expected criteria
func (s *SSLChecker) validateCertificate(certInfo *types.CertInfo, expected types.Expected) error {
	// Check certificate validity
	if !certInfo.IsValid {
		return fmt.Errorf("certificate is not valid (expired or not yet valid)")
	}
	
	// Check expiry days threshold
	if expected.CertExpiryDays > 0 && certInfo.DaysToExpiry <= expected.CertExpiryDays {
		return fmt.Errorf("certificate expires in %d days (threshold: %d days)", 
			certInfo.DaysToExpiry, expected.CertExpiryDays)
	}
	
	// Check valid domains if specified
	if len(expected.CertValidDomains) > 0 {
		validDomains := make(map[string]bool)
		
		// Add common name
		if certInfo.CommonName != "" {
			validDomains[certInfo.CommonName] = true
		}
		
		// Add DNS names
		for _, dnsName := range certInfo.DNSNames {
			validDomains[dnsName] = true
		}
		
		// Check if all expected domains are covered
		for _, expectedDomain := range expected.CertValidDomains {
			if !validDomains[expectedDomain] {
				// Check for wildcard match
				wildcardMatch := false
				for validDomain := range validDomains {
					if strings.HasPrefix(validDomain, "*.") {
						wildcard := strings.TrimPrefix(validDomain, "*.")
						if strings.HasSuffix(expectedDomain, wildcard) {
							wildcardMatch = true
							break
						}
					}
				}
				
				if !wildcardMatch {
					return fmt.Errorf("certificate does not cover expected domain: %s", expectedDomain)
				}
			}
		}
	}
	
	return nil
}