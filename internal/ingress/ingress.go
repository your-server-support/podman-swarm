package ingress

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/your-server-support/podman-swarm/internal/discovery"
	"github.com/your-server-support/podman-swarm/internal/types"
)

type IngressController struct {
	discovery     *discovery.Discovery
	logger        *logrus.Logger
	mu            sync.RWMutex
	rules         map[string]*types.Ingress
	router        *gin.Engine
	port          int
	proxies       map[string]*httputil.ReverseProxy
	roundRobinIdx map[string]int // serviceKey -> current index for round-robin
	localNodeName string
}

func NewIngressController(discovery *discovery.Discovery, port int, localNodeName string, logger *logrus.Logger) *IngressController {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	ic := &IngressController{
		discovery:     discovery,
		logger:        logger,
		port:          port,
		router:        router,
		rules:         make(map[string]*types.Ingress),
		proxies:       make(map[string]*httputil.ReverseProxy),
		roundRobinIdx: make(map[string]int),
		localNodeName: localNodeName,
	}

	// Setup catch-all route
	router.NoRoute(ic.handleRequest)

	return ic
}

// AddIngress adds an ingress rule
func (ic *IngressController) AddIngress(ingress *types.Ingress) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	key := fmt.Sprintf("%s/%s", ingress.Namespace, ingress.Name)
	ic.rules[key] = ingress

	ic.logger.Infof("Added ingress rule: %s", key)
	return nil
}

// RemoveIngress removes an ingress rule
func (ic *IngressController) RemoveIngress(namespace, name string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	delete(ic.rules, key)
	delete(ic.proxies, key)

	ic.logger.Infof("Removed ingress rule: %s", key)
}

// handleRequest handles incoming HTTP requests
func (ic *IngressController) handleRequest(c *gin.Context) {
	host := c.Request.Host
	path := c.Request.URL.Path

	ic.mu.RLock()
	defer ic.mu.RUnlock()

	// Find matching ingress rule
	var matchedIngress *types.Ingress
	var matchedPath *types.IngressPath

	for _, ingress := range ic.rules {
		for _, rule := range ingress.Rules {
			if rule.Host == host || rule.Host == "" {
				for _, p := range rule.Paths {
					if matchesPath(path, p.Path, p.PathType) {
						matchedIngress = ingress
						matchedPath = &p
						break
					}
				}
				if matchedPath != nil {
					break
				}
			}
		}
		if matchedPath != nil {
			break
		}
	}

	if matchedPath == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No ingress rule found"})
		return
	}

	// Discover service endpoints
	endpoints, err := ic.discovery.GetServiceEndpoints(matchedPath.ServiceName, matchedIngress.Namespace)
	if err != nil || len(endpoints) == 0 {
		ic.logger.Errorf("Service %s not found or no healthy instances", matchedPath.ServiceName)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	// Round-robin selection
	proxyKey := fmt.Sprintf("%s/%s", matchedIngress.Namespace, matchedPath.ServiceName)
	ic.mu.Lock()
	idx := ic.roundRobinIdx[proxyKey]
	selectedEndpoint := endpoints[idx%len(endpoints)]
	ic.roundRobinIdx[proxyKey] = (idx + 1) % len(endpoints)
	ic.mu.Unlock()

	// Determine target address
	var target string
	if selectedEndpoint.NodeName == ic.localNodeName {
		// Local pod - use localhost
		target = fmt.Sprintf("localhost:%d", selectedEndpoint.Port)
		ic.logger.Debugf("Routing to local pod: %s", target)
	} else {
		// Remote pod - need to get node address
		// For now, use the address from endpoint (which is node name)
		// In production, you'd resolve node name to IP or use node's actual address
		// Note: This assumes nodes are reachable by their names or addresses
		target = fmt.Sprintf("%s:%d", selectedEndpoint.Address, selectedEndpoint.Port)
		ic.logger.Debugf("Routing to remote pod on node %s: %s", selectedEndpoint.NodeName, target)
	}

	// Get or create reverse proxy
	proxy, ok := ic.proxies[proxyKey]
	if !ok || proxy == nil {
		targetURL, err := url.Parse(fmt.Sprintf("http://%s", target))
		if err != nil {
			ic.logger.Errorf("Invalid target URL %s: %v", target, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid target URL"})
			return
		}

		proxy = httputil.NewSingleHostReverseProxy(targetURL)
		ic.proxies[proxyKey] = proxy
		ic.logger.Infof("Created reverse proxy for %s -> %s", proxyKey, target)
	} else {
		// Update proxy target if endpoint changed
		// Note: In production, you might want to recreate proxy for each request
		// or use a more sophisticated load balancer
	}

	// Proxy the request
	proxy.ServeHTTP(c.Writer, c.Request)
}

// matchesPath checks if a request path matches an ingress path
func matchesPath(requestPath, ingressPath string, pathType *networkingv1.PathType) bool {
	if ingressPath == "" {
		return true
	}

	if pathType == nil {
		// Default to Prefix
		return len(requestPath) >= len(ingressPath) && requestPath[:len(ingressPath)] == ingressPath
	}

	switch *pathType {
	case networkingv1.PathTypeExact:
		return requestPath == ingressPath
	case networkingv1.PathTypePrefix:
		return len(requestPath) >= len(ingressPath) && requestPath[:len(ingressPath)] == ingressPath
	case networkingv1.PathTypeImplementationSpecific:
		return len(requestPath) >= len(ingressPath) && requestPath[:len(ingressPath)] == ingressPath
	default:
		return false
	}
}

// Start starts the ingress controller
func (ic *IngressController) Start() error {
	ic.logger.Infof("Starting ingress controller on port %d", ic.port)
	return ic.router.Run(fmt.Sprintf(":%d", ic.port))
}
