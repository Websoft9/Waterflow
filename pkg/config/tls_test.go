package config

import (
	"crypto/tls"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCert creates a temporary self-signed certificate for testing.
func generateTestCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	tmpDir := t.TempDir()
	certFile = filepath.Join(tmpDir, "test.crt")
	keyFile = filepath.Join(tmpDir, "test.key")

	// Use openssl to generate a real self-signed certificate
	// This requires openssl to be installed on the system
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
		"-keyout", keyFile,
		"-out", certFile,
		"-days", "1",
		"-subj", "/CN=localhost",
		"-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Skipping test: openssl not available or failed: %v\nOutput: %s", err, output)
	}

	return certFile, keyFile
}

func TestGetTLSCertFile(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		expected string
	}{
		{
			name: "new format takes priority",
			config: ServerConfig{
				TLSCertFile: "/old/path/cert.pem",
				HTTPS: HTTPSConfig{
					CertFile: "/new/path/cert.pem",
				},
			},
			expected: "/new/path/cert.pem",
		},
		{
			name: "fallback to old format",
			config: ServerConfig{
				TLSCertFile: "/old/path/cert.pem",
			},
			expected: "/old/path/cert.pem",
		},
		{
			name:     "empty when both are empty",
			config:   ServerConfig{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetTLSCertFile()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTLSKeyFile(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		expected string
	}{
		{
			name: "new format takes priority",
			config: ServerConfig{
				TLSKeyFile: "/old/path/key.pem",
				HTTPS: HTTPSConfig{
					KeyFile: "/new/path/key.pem",
				},
			},
			expected: "/new/path/key.pem",
		},
		{
			name: "fallback to old format",
			config: ServerConfig{
				TLSKeyFile: "/old/path/key.pem",
			},
			expected: "/old/path/key.pem",
		},
		{
			name:     "empty when both are empty",
			config:   ServerConfig{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetTLSKeyFile()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTLSEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		expected bool
	}{
		{
			name: "enabled with new format",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled: true,
				},
			},
			expected: true,
		},
		{
			name: "enabled with old format (both cert and key set)",
			config: ServerConfig{
				TLSCertFile: "/path/to/cert.pem",
				TLSKeyFile:  "/path/to/key.pem",
			},
			expected: true,
		},
		{
			name: "disabled when only cert is set",
			config: ServerConfig{
				TLSCertFile: "/path/to/cert.pem",
			},
			expected: false,
		},
		{
			name:     "disabled when nothing is set",
			config:   ServerConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsTLSEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateTLS(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	tests := []struct {
		name      string
		config    ServerConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid new format config",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled:       true,
					CertFile:      certFile,
					KeyFile:       keyFile,
					MinTLSVersion: "1.2",
				},
			},
			expectErr: false,
		},
		{
			name: "valid old format config",
			config: ServerConfig{
				TLSCertFile: certFile,
				TLSKeyFile:  keyFile,
			},
			expectErr: false,
		},
		{
			name: "missing cert file in new format",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled: true,
					KeyFile: keyFile,
				},
			},
			expectErr: true,
			errMsg:    "TLS certificate file is required",
		},
		{
			name: "missing key file in new format",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled:  true,
					CertFile: certFile,
				},
			},
			expectErr: true,
			errMsg:    "TLS key file is required",
		},
		{
			name: "cert file not found",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled:  true,
					CertFile: "/nonexistent/cert.pem",
					KeyFile:  keyFile,
				},
			},
			expectErr: true,
			errMsg:    "certificate file not found",
		},
		{
			name: "key file not found",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled:  true,
					CertFile: certFile,
					KeyFile:  "/nonexistent/key.pem",
				},
			},
			expectErr: true,
			errMsg:    "private key file not found",
		},
		{
			name: "invalid TLS version",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled:       true,
					CertFile:      certFile,
					KeyFile:       keyFile,
					MinTLSVersion: "0.9",
				},
			},
			expectErr: true,
			errMsg:    "invalid min_tls_version",
		},
		{
			name: "TLS disabled - no validation",
			config: ServerConfig{
				HTTPS: HTTPSConfig{
					Enabled: false,
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateTLS()
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMinTLSVersion(t *testing.T) {
	tests := []struct {
		name     string
		config   HTTPSConfig
		expected uint16
	}{
		{
			name: "TLS 1.0",
			config: HTTPSConfig{
				MinTLSVersion: "1.0",
			},
			expected: tls.VersionTLS10,
		},
		{
			name: "TLS 1.1",
			config: HTTPSConfig{
				MinTLSVersion: "1.1",
			},
			expected: tls.VersionTLS11,
		},
		{
			name: "TLS 1.2",
			config: HTTPSConfig{
				MinTLSVersion: "1.2",
			},
			expected: tls.VersionTLS12,
		},
		{
			name: "TLS 1.3",
			config: HTTPSConfig{
				MinTLSVersion: "1.3",
			},
			expected: tls.VersionTLS13,
		},
		{
			name:     "default to TLS 1.2",
			config:   HTTPSConfig{},
			expected: tls.VersionTLS12,
		},
		{
			name: "invalid version defaults to TLS 1.2",
			config: HTTPSConfig{
				MinTLSVersion: "invalid",
			},
			expected: tls.VersionTLS12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetMinTLSVersion()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         HTTPSConfig
		expectedMinVer uint16
	}{
		{
			name: "TLS 1.2 configuration",
			config: HTTPSConfig{
				MinTLSVersion: "1.2",
			},
			expectedMinVer: tls.VersionTLS12,
		},
		{
			name: "TLS 1.3 configuration",
			config: HTTPSConfig{
				MinTLSVersion: "1.3",
			},
			expectedMinVer: tls.VersionTLS13,
		},
		{
			name:           "default configuration",
			config:         HTTPSConfig{},
			expectedMinVer: tls.VersionTLS12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig := tt.config.BuildTLSConfig()

			assert.NotNil(t, tlsConfig)
			assert.Equal(t, tt.expectedMinVer, tlsConfig.MinVersion)
			// assert.True(t, tlsConfig.PreferServerCipherSuites) // Deprecated since Go 1.18
			// Verify recommended cipher suites are present
			expectedCiphers := []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			}
			assert.Equal(t, expectedCiphers, tlsConfig.CipherSuites)
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	t.Run("old config format works", func(t *testing.T) {
		cfg := ServerConfig{
			TLSCertFile: certFile,
			TLSKeyFile:  keyFile,
		}

		assert.True(t, cfg.IsTLSEnabled())
		assert.Equal(t, certFile, cfg.GetTLSCertFile())
		assert.Equal(t, keyFile, cfg.GetTLSKeyFile())
		assert.NoError(t, cfg.ValidateTLS())
	})

	t.Run("new config overrides old", func(t *testing.T) {
		newCert, newKey := generateTestCert(t)

		cfg := ServerConfig{
			TLSCertFile: certFile,
			TLSKeyFile:  keyFile,
			HTTPS: HTTPSConfig{
				Enabled:  true,
				CertFile: newCert,
				KeyFile:  newKey,
			},
		}

		assert.True(t, cfg.IsTLSEnabled())
		assert.Equal(t, newCert, cfg.GetTLSCertFile())
		assert.Equal(t, newKey, cfg.GetTLSKeyFile())
		assert.NoError(t, cfg.ValidateTLS())
	})

	t.Run("environment variables work with both formats", func(t *testing.T) {
		// Note: This test verifies the design supports both old and new env var prefixes
		// Actual env var loading is handled by Viper in config.Load()
		// which maps WATERFLOW_SERVER_TLS_CERT_FILE -> server.tls_cert_file
		// and WATERFLOW_SERVER_HTTPS_CERT_FILE -> server.https.cert_file

		// Old format via TLSCertFile field
		cfgOld := ServerConfig{
			TLSCertFile: certFile,
			TLSKeyFile:  keyFile,
		}
		assert.Equal(t, certFile, cfgOld.GetTLSCertFile())
		assert.Equal(t, keyFile, cfgOld.GetTLSKeyFile())

		// New format via HTTPS.CertFile field
		cfgNew := ServerConfig{
			HTTPS: HTTPSConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		}
		assert.Equal(t, certFile, cfgNew.GetTLSCertFile())
		assert.Equal(t, keyFile, cfgNew.GetTLSKeyFile())
	})
}
