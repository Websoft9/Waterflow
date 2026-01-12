// Package config provides TLS configuration and version mapping.
package config

import (
	"crypto/tls"
	"fmt"
	"os"
)

// HTTPConfig holds HTTP server configuration.
type HTTPConfig struct {
	// Enabled enables the HTTP server.
	Enabled bool `mapstructure:"enabled"`
	// Port is the HTTP listening port (default: 8080).
	Port int `mapstructure:"port"`
	// RedirectToHTTPS enables automatic redirect to HTTPS.
	RedirectToHTTPS bool `mapstructure:"redirect_to_https"`
}

// HTTPSConfig holds HTTPS/TLS server configuration.
type HTTPSConfig struct {
	// Enabled enables the HTTPS server.
	Enabled bool `mapstructure:"enabled"`
	// Port is the HTTPS listening port (default: 8443).
	Port int `mapstructure:"port"`
	// CertFile is the path to TLS certificate file.
	CertFile string `mapstructure:"cert_file"`
	// KeyFile is the path to TLS private key file.
	KeyFile string `mapstructure:"key_file"`
	// MinTLSVersion is the minimum TLS version ("1.0", "1.1", "1.2", "1.3", default: "1.2").
	MinTLSVersion string `mapstructure:"min_tls_version"`
}

// tlsVersionMap maps TLS version strings to crypto/tls constants.
var tlsVersionMap = map[string]uint16{
	"1.0": tls.VersionTLS10,
	"1.1": tls.VersionTLS11,
	"1.2": tls.VersionTLS12,
	"1.3": tls.VersionTLS13,
}

// GetTLSCertFile returns the TLS certificate file path.
// Priority: HTTPS.CertFile > TLSCertFile (backward compatibility).
func (s *ServerConfig) GetTLSCertFile() string {
	if s.HTTPS.CertFile != "" {
		return s.HTTPS.CertFile
	}
	return s.TLSCertFile
}

// GetTLSKeyFile returns the TLS private key file path.
// Priority: HTTPS.KeyFile > TLSKeyFile (backward compatibility).
func (s *ServerConfig) GetTLSKeyFile() string {
	if s.HTTPS.KeyFile != "" {
		return s.HTTPS.KeyFile
	}
	return s.TLSKeyFile
}

// IsTLSEnabled returns true if TLS is enabled (either old or new config format).
func (s *ServerConfig) IsTLSEnabled() bool {
	// New format: explicit HTTPS enabled flag
	if s.HTTPS.Enabled {
		return true
	}
	// Old format: implicit TLS if cert/key files are set
	if s.TLSCertFile != "" && s.TLSKeyFile != "" {
		return true
	}
	return false
}

// ValidateTLS validates TLS configuration (supports both old and new formats).
func (s *ServerConfig) ValidateTLS() error {
	// Check if TLS is enabled
	if !s.IsTLSEnabled() {
		return nil
	}

	// Get certificate file paths (prioritize new format)
	certFile := s.GetTLSCertFile()
	keyFile := s.GetTLSKeyFile()

	// Validate certificate file is set
	if certFile == "" {
		return fmt.Errorf("TLS certificate file is required (set server.https.cert_file or server.tls_cert_file)\n" +
			"  Set WATERFLOW_HTTPS_CERT_FILE=/path/to/cert.pem or WATERFLOW_SERVER_TLS_CERT_FILE=/path/to/cert.pem")
	}

	// Validate private key file is set
	if keyFile == "" {
		return fmt.Errorf("TLS key file is required (set server.https.key_file or server.tls_key_file)\n" +
			"  Set WATERFLOW_HTTPS_KEY_FILE=/path/to/key.pem or WATERFLOW_SERVER_TLS_KEY_FILE=/path/to/key.pem")
	}

	// Check certificate file exists
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file not found: %s\n"+
			"  Ensure the file exists or generate self-signed cert with: ./scripts/generate-self-signed-cert.sh", certFile)
	}

	// Check private key file exists
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return fmt.Errorf("private key file not found: %s\n"+
			"  Ensure the file exists or generate self-signed cert with: ./scripts/generate-self-signed-cert.sh", keyFile)
	}

	// Validate TLS version
	minTLSVersion := s.HTTPS.MinTLSVersion
	if minTLSVersion == "" {
		minTLSVersion = "1.2" // Default TLS 1.2
	}
	if _, ok := tlsVersionMap[minTLSVersion]; !ok {
		return fmt.Errorf("invalid min_tls_version: %s, must be one of: 1.0, 1.1, 1.2, 1.3\n"+
			"  Set WATERFLOW_HTTPS_MIN_TLS_VERSION=1.2 or update server.https.min_tls_version in config file", minTLSVersion)
	}

	// Try to load certificate to validate format
	_, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w\n"+
			"  Verify certificate and key files are valid PEM format", err)
	}

	return nil
}

// GetMinTLSVersion returns the minimum TLS version as crypto/tls constant.
// Returns TLS 1.2 as default if not specified.
func (h *HTTPSConfig) GetMinTLSVersion() uint16 {
	minVersion := h.MinTLSVersion
	if minVersion == "" {
		minVersion = "1.2" // Default to TLS 1.2
	}
	if version, ok := tlsVersionMap[minVersion]; ok {
		return version
	}
	// Fallback to TLS 1.2 if invalid (should be caught by ValidateTLS)
	return tls.VersionTLS12
}

// BuildTLSConfig creates a *tls.Config with secure defaults.
func (h *HTTPSConfig) BuildTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: h.GetMinTLSVersion(), //nolint:gosec // MinVersion is configurable
		// Recommended cipher suites (disable weak ciphers)
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
	}
}
