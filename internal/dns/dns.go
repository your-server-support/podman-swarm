package dns

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"

	"github.com/your-server-support/podman-swarm/internal/discovery"
)

const (
	// DefaultClusterDomain is the default cluster domain for service DNS
	DefaultClusterDomain = "cluster.local"
	// DefaultDNSPort is the default DNS server port
	DefaultDNSPort = 53
)

// Server represents a DNS server for service discovery
type Server struct {
	discovery     *discovery.Discovery
	logger        *logrus.Logger
	clusterDomain string
	port          int
	server        *dns.Server
	mu            sync.RWMutex
	localNodeIP   string
	upstreamDNS   []string // Upstream DNS servers for forwarding
	whitelist     *DNSWhitelist // DNS whitelist for external hosts
}

// DNSWhitelist represents DNS whitelist configuration
type DNSWhitelist struct {
	Enabled bool
	Hosts   map[string]bool // Use map for fast lookup
}

// NewServer creates a new DNS server
func NewServer(discovery *discovery.Discovery, clusterDomain string, port int, localNodeIP string, upstreamDNS []string, logger *logrus.Logger) *Server {
	if clusterDomain == "" {
		clusterDomain = DefaultClusterDomain
	}
	if port == 0 {
		port = DefaultDNSPort
	}
	if len(upstreamDNS) == 0 {
		// Default upstream DNS servers
		upstreamDNS = []string{"8.8.8.8:53", "8.8.4.4:53"}
	}

	return &Server{
		discovery:     discovery,
		logger:        logger,
		clusterDomain: clusterDomain,
		port:          port,
		localNodeIP:   localNodeIP,
		upstreamDNS:   upstreamDNS,
		whitelist: &DNSWhitelist{
			Enabled: false, // By default, allow all
			Hosts:   make(map[string]bool),
		},
	}
}

// Start starts the DNS server
func (s *Server) Start() error {
	// Handle all DNS queries (not just cluster domain)
	dns.HandleFunc(".", s.handleDNS)

	// Start UDP server
	udpServer := &dns.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Net:     "udp",
		Handler: dns.HandlerFunc(s.handleDNS),
	}

	// Start TCP server
	tcpServer := &dns.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Net:     "tcp",
		Handler: dns.HandlerFunc(s.handleDNS),
	}

	s.server = udpServer

	// Start both servers
	go func() {
		if err := udpServer.ListenAndServe(); err != nil {
			s.logger.Errorf("DNS UDP server error: %v", err)
		}
	}()

	go func() {
		if err := tcpServer.ListenAndServe(); err != nil {
			s.logger.Errorf("DNS TCP server error: %v", err)
		}
	}()

	s.logger.Infof("DNS server started on port %d for domain %s", s.port, s.clusterDomain)
	s.logger.Infof("Upstream DNS servers: %v", s.upstreamDNS)
	return nil
}

// Stop stops the DNS server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown()
	}
	return nil
}

// handleDNS handles DNS queries
func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	// Check if query is for cluster domain
	isClusterDomain := s.isClusterDomainQuery(r.Question)

	if isClusterDomain {
		// Handle cluster domain queries locally
		s.handleClusterQuery(w, r)
	} else {
		// Forward to upstream DNS servers
		s.forwardQuery(w, r)
	}
}

// isClusterDomainQuery checks if any question is for the cluster domain
func (s *Server) isClusterDomainQuery(questions []dns.Question) bool {
	for _, q := range questions {
		name := strings.ToLower(strings.TrimSuffix(q.Name, "."))
		if strings.HasSuffix(name, "."+s.clusterDomain) || name == s.clusterDomain {
			return true
		}
	}
	return false
}

// handleClusterQuery handles queries for cluster domain
func (s *Server) handleClusterQuery(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		s.logger.Debugf("Cluster DNS query: %s (type: %s)", q.Name, dns.TypeToString[q.Qtype])

		switch q.Qtype {
		case dns.TypeA:
			s.handleAQuery(m, q)
		case dns.TypeSRV:
			s.handleSRVQuery(m, q)
		case dns.TypeAAAA:
			// IPv6 not supported yet, return empty
			m.Answer = append(m.Answer, s.emptyAnswer(q))
		default:
			// Return empty answer for unsupported types
			m.Answer = append(m.Answer, s.emptyAnswer(q))
		}
	}

	w.WriteMsg(m)
}

// forwardQuery forwards DNS queries to upstream DNS servers
func (s *Server) forwardQuery(w dns.ResponseWriter, r *dns.Msg) {
	queryName := ""
	if len(r.Question) > 0 {
		queryName = r.Question[0].Name
	}

	// Check whitelist if enabled
	if s.isWhitelistEnabled() {
		if !s.isHostAllowed(queryName) {
			s.logger.Warnf("DNS query for %s blocked by whitelist", queryName)
			m := new(dns.Msg)
			m.SetReply(r)
			m.Rcode = dns.RcodeRefused
			w.WriteMsg(m)
			return
		}
	}

	s.logger.Debugf("Forwarding DNS query: %s to upstream servers", queryName)

	// Try each upstream DNS server
	var lastErr error
	for _, upstream := range s.upstreamDNS {
		// Try UDP first (faster for most queries)
		client := &dns.Client{
			Net:     "udp",
			Timeout: 5 * time.Second,
		}

		resp, rtt, err := client.Exchange(r, upstream)
		if err != nil {
			s.logger.Debugf("Failed to forward UDP query to %s: %v, trying TCP", upstream, err)
			// Try TCP if UDP fails
			client.Net = "tcp"
			resp, rtt, err = client.Exchange(r, upstream)
			if err != nil {
				s.logger.Debugf("Failed to forward TCP query to %s: %v", upstream, err)
				lastErr = err
				continue
			}
		}

		if resp != nil {
			if resp.Rcode == dns.RcodeSuccess {
				// Check CNAME records in response if whitelist is enabled
				if s.isWhitelistEnabled() {
					if !s.validateCNAMERecords(resp) {
						s.logger.Warnf("DNS response for %s contains CNAME to non-whitelisted domain, blocking", queryName)
						lastErr = fmt.Errorf("CNAME target not in whitelist")
						continue // Try next upstream server
					}
				}

				s.logger.Debugf("Successfully forwarded query %s to %s (RTT: %v)", queryName, upstream, rtt)
				w.WriteMsg(resp)
				return
			}
			// If we got a response but it's not successful, try next server
			s.logger.Debugf("Upstream %s returned RCODE %d for query %s", upstream, resp.Rcode, queryName)
			lastErr = fmt.Errorf("upstream returned RCODE %d", resp.Rcode)
		}
	}

	// If all upstream servers failed, return SERVFAIL
	s.logger.Warnf("All upstream DNS servers failed for query %s, last error: %v", queryName, lastErr)
	m := new(dns.Msg)
	m.SetReply(r)
	m.Rcode = dns.RcodeServerFailure
	w.WriteMsg(m)
}

// handleAQuery handles A record queries
// Format: service-name.namespace.cluster.local
// Format: service-name.namespace.svc.cluster.local (Kubernetes compatible)
func (s *Server) handleAQuery(m *dns.Msg, q dns.Question) {
	// Parse the query name
	// Examples:
	// - postgres-service.default.cluster.local
	// - postgres-service.default.svc.cluster.local
	// - postgres-service.default
	serviceName, namespace, err := s.parseServiceName(q.Name)
	if err != nil {
		s.logger.Debugf("Failed to parse service name %s: %v", q.Name, err)
		return
	}

	// Get service endpoints from discovery
	endpoints, err := s.discovery.GetServiceEndpoints(serviceName, namespace)
	if err != nil {
		s.logger.Debugf("Service %s.%s not found: %v", serviceName, namespace, err)
		return
	}

	// Add A records for each healthy endpoint
	for _, endpoint := range endpoints {
		// Use the node address (IP) for the A record
		rr, err := dns.NewRR(fmt.Sprintf("%s %d IN A %s", q.Name, 60, endpoint.Address))
		if err != nil {
			s.logger.Warnf("Failed to create A record: %v", err)
			continue
		}
		m.Answer = append(m.Answer, rr)
	}

	if len(m.Answer) > 0 {
		s.logger.Debugf("Resolved %s to %d endpoints", q.Name, len(m.Answer))
	}
}

// handleSRVQuery handles SRV record queries
// Format: _port-name._protocol.service-name.namespace.cluster.local
// Format: _port-name._protocol.service-name.namespace.svc.cluster.local
func (s *Server) handleSRVQuery(m *dns.Msg, q dns.Question) {
	// Parse SRV query
	// Example: _http._tcp.postgres-service.default.cluster.local
	portName, protocol, serviceName, namespace, err := s.parseSRVName(q.Name)
	if err != nil {
		s.logger.Debugf("Failed to parse SRV name %s: %v", q.Name, err)
		return
	}

	// Get service endpoints
	endpoints, err := s.discovery.GetServiceEndpoints(serviceName, namespace)
	if err != nil {
		s.logger.Debugf("Service %s.%s not found for SRV query (port: %s, protocol: %s): %v", serviceName, namespace, portName, protocol, err)
		return
	}

	// Find port by name or use first port
	var targetPort int32
	for _, endpoint := range endpoints {
		// For now, use the endpoint port
		// In a full implementation, we'd match portName to named ports
		targetPort = endpoint.Port
		break
	}
	s.logger.Debugf("SRV query for %s.%s (port: %s, protocol: %s) -> using port %d", serviceName, namespace, portName, protocol, targetPort)

	// Add SRV records for each endpoint
	priority := uint16(10)
	weight := uint16(10)
	for i, endpoint := range endpoints {
		// Create target name: service-name.namespace.cluster.local
		target := fmt.Sprintf("%s.%s.%s.", serviceName, namespace, s.clusterDomain)
		
		rr, err := dns.NewRR(fmt.Sprintf("%s %d IN SRV %d %d %d %s",
			q.Name, 60, priority, weight, targetPort, target))
		if err != nil {
			s.logger.Warnf("Failed to create SRV record: %v", err)
			continue
		}
		m.Answer = append(m.Answer, rr)

		// Also add A record for the target
		aRR, err := dns.NewRR(fmt.Sprintf("%s %d IN A %s", target, 60, endpoint.Address))
		if err == nil {
			m.Extra = append(m.Extra, aRR)
		}

		// Round-robin: adjust priority/weight for load balancing
		if i > 0 {
			priority += 10
		}
	}

	if len(m.Answer) > 0 {
		s.logger.Debugf("Resolved SRV %s to %d endpoints", q.Name, len(m.Answer))
	}
}

// parseServiceName parses a DNS name to extract service name and namespace
// Examples:
// - postgres-service.default.cluster.local -> (postgres-service, default)
// - postgres-service.default.svc.cluster.local -> (postgres-service, default)
// - postgres-service.default -> (postgres-service, default)
func (s *Server) parseServiceName(name string) (serviceName, namespace string, err error) {
	// Remove trailing dot
	name = strings.TrimSuffix(name, ".")

	// Remove cluster domain
	name = strings.TrimSuffix(name, "."+s.clusterDomain)
	name = strings.TrimSuffix(name, ".svc."+s.clusterDomain)

	// Split by dots
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid service name format: %s", name)
	}

	// Last part is namespace, everything before is service name
	namespace = parts[len(parts)-1]
	serviceName = strings.Join(parts[:len(parts)-1], ".")

	return serviceName, namespace, nil
}

// parseSRVName parses an SRV DNS name
// Example: _http._tcp.postgres-service.default.cluster.local
// Returns: (http, tcp, postgres-service, default, nil)
func (s *Server) parseSRVName(name string) (portName, protocol, serviceName, namespace string, err error) {
	// Remove trailing dot
	name = strings.TrimSuffix(name, ".")

	// Remove cluster domain
	name = strings.TrimSuffix(name, "."+s.clusterDomain)
	name = strings.TrimSuffix(name, ".svc."+s.clusterDomain)

	// Split by dots
	parts := strings.Split(name, ".")
	if len(parts) < 4 {
		return "", "", "", "", fmt.Errorf("invalid SRV name format: %s", name)
	}

	// Format: _port-name._protocol.service-name.namespace
	// Parts: [0] = _port-name, [1] = _protocol, [2:] = service-name parts, [last] = namespace
	if !strings.HasPrefix(parts[0], "_") || !strings.HasPrefix(parts[1], "_") {
		return "", "", "", "", fmt.Errorf("invalid SRV name format: %s", name)
	}

	portName = strings.TrimPrefix(parts[0], "_")
	protocol = strings.TrimPrefix(parts[1], "_")
	namespace = parts[len(parts)-1]
	serviceName = strings.Join(parts[2:len(parts)-1], ".")

	return portName, protocol, serviceName, namespace, nil
}

// emptyAnswer returns an empty answer for unsupported queries
func (s *Server) emptyAnswer(q dns.Question) dns.RR {
	rr, _ := dns.NewRR(fmt.Sprintf("%s 0 IN A 0.0.0.0", q.Name))
	return rr
}

// GetDNSAddress returns the DNS server address that should be used by containers
func (s *Server) GetDNSAddress() string {
	// Return the local node IP with DNS port
	// Containers should use this as their DNS server
	if s.localNodeIP != "" {
		return fmt.Sprintf("%s:%d", s.localNodeIP, s.port)
	}
	// Fallback to localhost
	return fmt.Sprintf("127.0.0.1:%d", s.port)
}

// GetDNSIP returns just the IP address (without port) for DNS server
func (s *Server) GetDNSIP() string {
	if s.localNodeIP != "" {
		return s.localNodeIP
	}
	return "127.0.0.1"
}

// SetWhitelist sets the DNS whitelist configuration
func (s *Server) SetWhitelist(enabled bool, hosts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.whitelist.Enabled = enabled
	s.whitelist.Hosts = make(map[string]bool)

	// Normalize and add hosts to map
	for _, host := range hosts {
		// Remove trailing dot and convert to lowercase
		normalized := strings.ToLower(strings.TrimSuffix(host, "."))
		// Also add domain without subdomains for wildcard-like matching
		s.whitelist.Hosts[normalized] = true
		// Add with trailing dot for exact match
		s.whitelist.Hosts[normalized+"."] = true
	}

	s.logger.Infof("DNS whitelist updated: enabled=%v, hosts=%d", enabled, len(hosts))
}

// GetWhitelist returns the current whitelist configuration
func (s *Server) GetWhitelist() (bool, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hosts := make([]string, 0, len(s.whitelist.Hosts))
	seen := make(map[string]bool)
	for host := range s.whitelist.Hosts {
		// Remove trailing dot for response
		normalized := strings.TrimSuffix(host, ".")
		if !seen[normalized] {
			hosts = append(hosts, normalized)
			seen[normalized] = true
		}
	}

	return s.whitelist.Enabled, hosts
}

// isWhitelistEnabled checks if whitelist is enabled
func (s *Server) isWhitelistEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.whitelist.Enabled
}

// isHostAllowed checks if a host is allowed by whitelist
func (s *Server) isHostAllowed(queryName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.whitelist.Enabled {
		return true // Allow all if whitelist is disabled
	}

	// Normalize query name
	normalized := strings.ToLower(strings.TrimSuffix(queryName, "."))

	// Check exact match
	if s.whitelist.Hosts[normalized] || s.whitelist.Hosts[normalized+"."] {
		return true
	}

	// Check subdomain matches (e.g., api.example.com matches example.com)
	parts := strings.Split(normalized, ".")
	for i := 0; i < len(parts); i++ {
		domain := strings.Join(parts[i:], ".")
		if s.whitelist.Hosts[domain] || s.whitelist.Hosts[domain+"."] {
			return true
		}
	}

	return false
}

// validateCNAMERecords validates all CNAME records in DNS response against whitelist
func (s *Server) validateCNAMERecords(resp *dns.Msg) bool {
	// Check CNAME records in Answer section
	for _, rr := range resp.Answer {
		if cname, ok := rr.(*dns.CNAME); ok {
			target := cname.Target
			if !s.isHostAllowed(target) {
				s.logger.Debugf("CNAME target %s is not in whitelist", target)
				return false
			}
		}
	}

	// Check CNAME records in Extra section (additional records)
	for _, rr := range resp.Extra {
		if cname, ok := rr.(*dns.CNAME); ok {
			target := cname.Target
			if !s.isHostAllowed(target) {
				s.logger.Debugf("CNAME target %s is not in whitelist", target)
				return false
			}
		}
	}

	// Also check for CNAME chains - follow CNAME records recursively
	// This handles cases where CNAME points to another CNAME
	cnameTargets := s.extractCNAMETargets(resp)
	for _, target := range cnameTargets {
		if !s.isHostAllowed(target) {
			s.logger.Debugf("CNAME chain target %s is not in whitelist", target)
			return false
		}
	}

	return true
}

// extractCNAMETargets extracts all CNAME targets from DNS response, including chains
func (s *Server) extractCNAMETargets(resp *dns.Msg) []string {
	targets := make([]string, 0)
	seen := make(map[string]bool)

	// Extract from Answer section
	for _, rr := range resp.Answer {
		if cname, ok := rr.(*dns.CNAME); ok {
			target := strings.ToLower(strings.TrimSuffix(cname.Target, "."))
			if !seen[target] {
				targets = append(targets, target)
				seen[target] = true
			}
		}
	}

	// Extract from Extra section
	for _, rr := range resp.Extra {
		if cname, ok := rr.(*dns.CNAME); ok {
			target := strings.ToLower(strings.TrimSuffix(cname.Target, "."))
			if !seen[target] {
				targets = append(targets, target)
				seen[target] = true
			}
		}
	}

	return targets
}
