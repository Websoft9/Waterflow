package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/config"
	"github.com/Websoft9/waterflow/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create test config
func newTestConfig(port int) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            port,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
		Events: config.EventsConfig{
			HandlerType: "noop",
		},
	}
}

// Helper function to wait for server ready
func waitForServerReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:gosec // Test URL
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return context.DeadlineExceeded
}

// Helper function to wait for HTTPS server ready (skips TLS verification)
func waitForHTTPSServerReady(url string, timeout time.Duration) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Test client
		},
		Timeout: 1 * time.Second,
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return context.DeadlineExceeded
}

// generateTestCerts generates self-signed TLS certificate and key for testing
func generateTestCerts(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Waterflow Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Write certificate to temp file
	certPath := filepath.Join(t.TempDir(), "cert.pem")
	certOut, err := os.Create(certPath) //nolint:gosec // Test file
	require.NoError(t, err)
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, err)
	_ = certOut.Close()

	// Write private key to temp file
	keyPath := filepath.Join(t.TempDir(), "key.pem")
	keyOut, err := os.Create(keyPath) //nolint:gosec // Test file
	require.NoError(t, err)
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	require.NoError(t, err)
	_ = keyOut.Close()

	return certPath, keyPath
}

func TestServerStartAndShutdown(t *testing.T) {
	// Initialize logger for tests
	err := logger.Init("info", "json")
	require.NoError(t, err)

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            18080,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
	}

	// Create server
	srv := New(cfg, logger.Log, "v1.0.0-test", "abc123", "2025-12-19")
	assert.NotNil(t, srv)

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:18080/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	if err := resp.Body.Close(); err != nil {
		t.Logf("failed to close response body: %v", err)
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	assert.NoError(t, err)

	// Verify server stopped
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Server did not stop in time")
	}
}

func TestHealthEndpoint(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            18081,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
	}

	srv := New(cfg, logger.Log, "v1.0.0-test", "abc123", "2025-12-19")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:18081/health")
	require.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Logf("server shutdown failed: %v", err)
	}
}

// TestNew tests server initialization with different configurations
func TestNew(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	t.Run("basic initialization", func(t *testing.T) {
		cfg := newTestConfig(18082)
		srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

		assert.NotNil(t, srv)
		assert.Equal(t, "v1.0.0", srv.version)
		assert.Equal(t, "abc123", srv.commit)
		assert.Equal(t, "2025-01-01", srv.buildTime)
		assert.NotNil(t, srv.config)
		assert.NotNil(t, srv.logger)
		assert.NotNil(t, srv.eventDispatcher)
		assert.Nil(t, srv.temporalClient) // Not initialized until Start
	})

	t.Run("with webhook event handler", func(t *testing.T) {
		cfg := newTestConfig(18083)
		cfg.Events.HandlerType = "webhook"
		cfg.Events.Webhook.URL = "http://localhost:9999/webhook"
		cfg.Events.Webhook.Timeout = 5 * time.Second

		srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")
		assert.NotNil(t, srv)
		assert.NotNil(t, srv.eventDispatcher)
	})

	t.Run("with empty webhook URL", func(t *testing.T) {
		cfg := newTestConfig(18084)
		cfg.Events.HandlerType = "webhook"
		cfg.Events.Webhook.URL = "" // Empty URL should fallback to noop

		srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")
		assert.NotNil(t, srv)
		assert.NotNil(t, srv.eventDispatcher)
	})

	t.Run("with unknown event handler type", func(t *testing.T) {
		cfg := newTestConfig(18085)
		cfg.Events.HandlerType = "unknown"

		srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")
		assert.NotNil(t, srv)
		assert.NotNil(t, srv.eventDispatcher)
	})
}

// TestAgentMonitor tests the AgentMonitor lifecycle
func TestAgentMonitor(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	t.Run("start and stop", func(t *testing.T) {
		// AgentMonitor can be created with nil client for testing
		am := NewAgentMonitor(nil, logger.Log, 100*time.Millisecond)
		assert.NotNil(t, am)

		// Start monitoring
		am.Start()

		// Let it run for a bit
		time.Sleep(250 * time.Millisecond)

		// Stop should not panic
		am.Stop()
	})

	t.Run("stop immediately after start", func(t *testing.T) {
		am := NewAgentMonitor(nil, logger.Log, 1*time.Second)
		am.Start()
		am.Stop()
		// Should not panic or block
	})
}

// TestReadyEndpoint tests the ready endpoint
func TestReadyEndpoint(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	cfg := newTestConfig(18086)
	srv := New(cfg, logger.Log, "v1.0.0-test", "abc123", "2025-12-19")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	require.NoError(t, waitForServerReady("http://localhost:18086/health", 2*time.Second))

	resp, err := http.Get("http://localhost:18086/ready")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// TestVersionEndpoint tests the version endpoint
func TestVersionEndpoint(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	cfg := newTestConfig(18087)
	srv := New(cfg, logger.Log, "v1.2.3", "def456", "2025-12-20")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	require.NoError(t, waitForServerReady("http://localhost:18087/health", 2*time.Second))

	resp, err := http.Get("http://localhost:18087/version")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// TestShutdownWithNilComponents tests graceful shutdown when components are nil
func TestShutdownWithNilComponents(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	cfg := newTestConfig(18088)
	srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

	// Shutdown without starting (no httpServer)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestServerWithAuditLogging tests server with audit logging enabled
func TestServerWithAuditLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping audit test in short mode")
	}

	err := logger.Init("info", "json")
	require.NoError(t, err)

	tmpDir := t.TempDir()

	cfg := newTestConfig(18089)
	cfg.Audit.Enabled = true
	cfg.Audit.File.Path = tmpDir
	cfg.Audit.File.MaxSize = 10
	cfg.Audit.File.MaxAge = 7
	cfg.Audit.File.MaxBackups = 3

	srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	require.NoError(t, waitForServerReady("http://localhost:18089/health", 2*time.Second))

	// Make a request to trigger audit
	resp, err := http.Get("http://localhost:18089/health")
	require.NoError(t, err)
	_ = resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// TestMultipleEndpoints tests multiple API endpoints
func TestMultipleEndpoints(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	cfg := newTestConfig(18090)
	srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	require.NoError(t, waitForServerReady("http://localhost:18090/health", 2*time.Second))

	endpoints := []struct {
		path       string
		wantStatus int
	}{
		{"/health", http.StatusOK},
		{"/ready", http.StatusOK},
		{"/version", http.StatusOK},
		{"/v1/nodes", http.StatusOK},
		{"/nonexistent", http.StatusNotFound},
	}

	for _, tc := range endpoints {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get("http://localhost:18090" + tc.path)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// TestServerStartWithAuditLoggerError tests server start when audit logger fails
func TestServerStartWithAuditLoggerError(t *testing.T) {
	err := logger.Init("info", "json")
	require.NoError(t, err)

	cfg := newTestConfig(18091)
	cfg.Audit.Enabled = true
	cfg.Audit.File.Path = "/nonexistent/path/that/should/fail"

	srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	// Server should still start even if audit logger fails
	require.NoError(t, waitForServerReady("http://localhost:18091/health", 2*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// TestHTTPSServer tests HTTPS server functionality
func TestHTTPSServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HTTPS test in short mode")
	}

	err := logger.Init("info", "json")
	require.NoError(t, err)

	certFile, keyFile := generateTestCerts(t)

	cfg := newTestConfig(18092)
	cfg.Server.HTTPS.Enabled = true
	cfg.Server.HTTPS.Port = 18492
	cfg.Server.HTTPS.CertFile = certFile
	cfg.Server.HTTPS.KeyFile = keyFile
	cfg.Server.HTTPS.MinTLSVersion = "1.2"

	srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	// Wait for HTTPS server to be ready
	require.NoError(t, waitForHTTPSServerReady("https://localhost:18492/health", 3*time.Second))

	// Create client that skips TLS verification (self-signed cert)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // Test client
		},
	}

	resp, err := client.Get("https://localhost:18492/health")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// TestHTTPSWithHTTPRedirect tests HTTP to HTTPS redirect
func TestHTTPSWithHTTPRedirect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HTTP redirect test in short mode")
	}

	err := logger.Init("info", "json")
	require.NoError(t, err)

	certFile, keyFile := generateTestCerts(t)

	cfg := newTestConfig(18093)
	cfg.Server.HTTPS.Enabled = true
	cfg.Server.HTTPS.Port = 18493
	cfg.Server.HTTPS.CertFile = certFile
	cfg.Server.HTTPS.KeyFile = keyFile
	cfg.Server.HTTP.Enabled = true
	cfg.Server.HTTP.Port = 18193
	cfg.Server.HTTP.RedirectToHTTPS = true

	srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	// Wait for servers to be ready
	require.NoError(t, waitForHTTPSServerReady("https://localhost:18493/health", 3*time.Second))
	time.Sleep(100 * time.Millisecond) // Give HTTP redirect server time to start

	// Create client that does not follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Test that HTTP redirects to HTTPS
	resp, err := client.Get("http://localhost:18193/health")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "https://")
	assert.Contains(t, location, "18493")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// TestHTTPParallelWithHTTPS tests HTTP running parallel with HTTPS (no redirect)
func TestHTTPParallelWithHTTPS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel HTTP test in short mode")
	}

	err := logger.Init("info", "json")
	require.NoError(t, err)

	certFile, keyFile := generateTestCerts(t)

	cfg := newTestConfig(18094)
	cfg.Server.HTTPS.Enabled = true
	cfg.Server.HTTPS.Port = 18494
	cfg.Server.HTTPS.CertFile = certFile
	cfg.Server.HTTPS.KeyFile = keyFile
	cfg.Server.HTTP.Enabled = true
	cfg.Server.HTTP.Port = 18194
	cfg.Server.HTTP.RedirectToHTTPS = false // No redirect, serve content on HTTP

	srv := New(cfg, logger.Log, "v1.0.0", "abc123", "2025-01-01")

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	// Wait for HTTPS server
	require.NoError(t, waitForHTTPSServerReady("https://localhost:18494/health", 3*time.Second))
	time.Sleep(100 * time.Millisecond)

	// Test HTTP serves content directly
	resp, err := http.Get("http://localhost:18194/health")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
