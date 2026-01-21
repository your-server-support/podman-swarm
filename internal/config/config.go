package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	NodeName      string
	BindAddr      string
	APIAddr       string
	PodmanSocket  string
	DataDir       string
	JoinAddrs     []string
	JoinToken     string
	EncryptionKey string
	TLSCertFile   string
	TLSKeyFile    string
	TLSCAFile     string
	TLSSkipVerify bool
	IngressPort      int
	EnableIngress    bool
	DNSPort          int
	ClusterDomain    string
	UpstreamDNS      []string // Upstream DNS servers for forwarding non-cluster queries
}

func Load() *Config {
	cfg := &Config{}
	var joinStr string

	flag.StringVar(&cfg.NodeName, "node-name", getEnv("NODE_NAME", "node-1"), "Node name")
	flag.StringVar(&cfg.BindAddr, "bind-addr", getEnv("BIND_ADDR", "0.0.0.0:7946"), "Cluster bind address")
	flag.StringVar(&cfg.APIAddr, "api-addr", getEnv("API_ADDR", "0.0.0.0:8080"), "API server address")
	flag.StringVar(&cfg.PodmanSocket, "podman-socket", getEnv("PODMAN_SOCKET", "unix:///run/podman/podman.sock"), "Podman socket")
	flag.StringVar(&cfg.DataDir, "data-dir", getEnv("DATA_DIR", "/var/lib/podman-swarm"), "Data directory")
	flag.StringVar(&joinStr, "join", getEnv("JOIN", ""), "Comma-separated list of addresses to join")
	flag.StringVar(&cfg.JoinToken, "join-token", getEnv("JOIN_TOKEN", ""), "Join token for cluster authentication")
	flag.StringVar(&cfg.EncryptionKey, "encryption-key", getEnv("ENCRYPTION_KEY", ""), "Encryption key for cluster communication")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", getEnv("TLS_CERT", ""), "TLS certificate file")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", getEnv("TLS_KEY", ""), "TLS key file")
	flag.StringVar(&cfg.TLSCAFile, "tls-ca", getEnv("TLS_CA", ""), "TLS CA certificate file")
	flag.BoolVar(&cfg.TLSSkipVerify, "tls-skip-verify", getEnvBool("TLS_SKIP_VERIFY", false), "Skip TLS certificate verification")
	flag.IntVar(&cfg.IngressPort, "ingress-port", 80, "Ingress port")
	flag.BoolVar(&cfg.EnableIngress, "enable-ingress", true, "Enable ingress controller")
	flag.IntVar(&cfg.DNSPort, "dns-port", getEnvInt("DNS_PORT", 53), "DNS server port")
	flag.StringVar(&cfg.ClusterDomain, "cluster-domain", getEnv("CLUSTER_DOMAIN", "cluster.local"), "Cluster domain for DNS")
	var upstreamDNSStr string
	flag.StringVar(&upstreamDNSStr, "upstream-dns", getEnv("UPSTREAM_DNS", "8.8.8.8:53,8.8.4.4:53"), "Comma-separated list of upstream DNS servers (IP:port)")

	flag.Parse()

	// Parse join addresses
	if joinStr != "" {
		cfg.JoinAddrs = strings.Split(joinStr, ",")
		for i, addr := range cfg.JoinAddrs {
			cfg.JoinAddrs[i] = strings.TrimSpace(addr)
		}
	}

	// Parse upstream DNS servers
	if upstreamDNSStr != "" {
		cfg.UpstreamDNS = strings.Split(upstreamDNSStr, ",")
		for i, addr := range cfg.UpstreamDNS {
			cfg.UpstreamDNS[i] = strings.TrimSpace(addr)
			// Add default port if not specified
			if !strings.Contains(cfg.UpstreamDNS[i], ":") {
				cfg.UpstreamDNS[i] = cfg.UpstreamDNS[i] + ":53"
			}
		}
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
