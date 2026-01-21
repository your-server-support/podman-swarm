package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/sirupsen/logrus"
)

// TLSConfig holds TLS configuration
type TLSConfig struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	SkipVerify bool
}

// LoadTLSConfig loads TLS configuration from files
func LoadTLSConfig(cfg *TLSConfig, logger *logrus.Logger) (*tls.Config, error) {
	if cfg == nil || cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("TLS certificate and key files are required")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		ServerName:   "",
	}

	// Load CA certificate if provided
	if cfg.CAFile != "" {
		caCert, err := ioutil.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		// Use system CA pool
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			logger.Warnf("Failed to load system CA pool: %v", err)
			caCertPool = x509.NewCertPool()
		}
		tlsConfig.RootCAs = caCertPool
	}

	if cfg.SkipVerify {
		tlsConfig.InsecureSkipVerify = true
		logger.Warn("TLS certificate verification is disabled")
	}

	return tlsConfig, nil
}

// WrapConn wraps a connection with TLS
func WrapConn(conn net.Conn, tlsConfig *tls.Config, isServer bool) (net.Conn, error) {
	if tlsConfig == nil {
		return conn, nil
	}

	if isServer {
		return tls.Server(conn, tlsConfig), nil
	}
	return tls.Client(conn, tlsConfig), nil
}

// GenerateSelfSignedCert generates a self-signed certificate for testing
// In production, use proper CA-signed certificates
func GenerateSelfSignedCert(host string) (*tls.Certificate, error) {
	// This is a placeholder - in production, use proper certificate generation
	// For now, we'll use a simple approach with crypto/tls
	// In real implementation, use crypto/x509 to generate proper certificates
	
	// Note: This is simplified - for production use proper certificate generation
	// or tools like cfssl, openssl, etc.
	return nil, fmt.Errorf("self-signed certificate generation not implemented - use proper certificates")
}
