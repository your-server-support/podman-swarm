package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/your-server-support/podman-swarm/internal/cluster"
	"github.com/your-server-support/podman-swarm/internal/discovery"
	"github.com/your-server-support/podman-swarm/internal/dns"
	"github.com/your-server-support/podman-swarm/internal/ingress"
	"github.com/your-server-support/podman-swarm/internal/parser"
	"github.com/your-server-support/podman-swarm/internal/podman"
	"github.com/your-server-support/podman-swarm/internal/scheduler"
	"github.com/your-server-support/podman-swarm/internal/security"
	"github.com/your-server-support/podman-swarm/internal/types"
)

type API struct {
	parser       *parser.Parser
	scheduler    *scheduler.Scheduler
	podman       *podman.Client
	discovery    *discovery.Discovery
	ingress      *ingress.IngressController
	cluster      *cluster.Cluster
	dns          *dns.Server
	logger       *logrus.Logger
	deployments  map[string]*types.Deployment
	services     map[string]*types.Service
	ingresses    map[string]*types.Ingress
	tokenManager *security.APITokenManager
}

func NewAPI(
	parser *parser.Parser,
	scheduler *scheduler.Scheduler,
	podman *podman.Client,
	discovery *discovery.Discovery,
	ingress *ingress.IngressController,
	cluster *cluster.Cluster,
	dns *dns.Server,
	tokenManager *security.APITokenManager,
	logger *logrus.Logger,
) *API {
	return &API{
		parser:       parser,
		scheduler:    scheduler,
		podman:       podman,
		discovery:    discovery,
		ingress:      ingress,
		cluster:      cluster,
		dns:          dns,
		tokenManager: tokenManager,
		logger:       logger,
		deployments:  make(map[string]*types.Deployment),
		services:     make(map[string]*types.Service),
		ingresses:    make(map[string]*types.Ingress),
	}
}

func (a *API) SetupRoutes(router *gin.Engine, authEnabled bool) {
	// Apply authentication middleware
	router.Use(AuthMiddleware(a.tokenManager, authEnabled))

	v1 := router.Group("/api/v1")
	{
		v1.POST("/manifests", a.ApplyManifest)
		v1.DELETE("/manifests/:namespace/:name", a.DeleteManifest)
		v1.GET("/pods", a.ListPods)
		v1.GET("/pods/:namespace/:name", a.GetPod)
		v1.GET("/deployments", a.ListDeployments)
		v1.GET("/deployments/:namespace/:name", a.GetDeployment)
		v1.GET("/services", a.ListServices)
		v1.GET("/services/:namespace/:name/endpoints", a.GetServiceEndpoints)
		v1.GET("/services/:namespace/:name/addresses", a.GetServiceAddresses)
		v1.GET("/nodes", a.ListNodes)
		v1.GET("/health", a.Health)
		// DNS whitelist endpoints
		v1.GET("/dns/whitelist", a.GetDNSWhitelist)
		v1.PUT("/dns/whitelist", a.SetDNSWhitelist)
		v1.POST("/dns/whitelist/hosts", a.AddDNSWhitelistHost)
		v1.DELETE("/dns/whitelist/hosts/:host", a.RemoveDNSWhitelistHost)
		// API Token management endpoints
		v1.POST("/tokens", a.GenerateAPIToken)
		v1.GET("/tokens", a.ListAPITokens)
		v1.DELETE("/tokens/:token", a.RevokeAPIToken)
	}
}

func (a *API) ApplyManifest(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to read request body"})
		return
	}

	objects, err := a.parser.ParseManifest(body)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to parse manifest: %v", err)})
		return
	}

	for _, obj := range objects {
		switch o := obj.(type) {
		case *appsv1.Deployment:
			if err := a.applyDeployment(o); err != nil {
				a.logger.Errorf("Failed to apply deployment: %v", err)
				c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to apply deployment: %v", err)})
				return
			}
		case *corev1.Service:
			if err := a.applyService(o); err != nil {
				a.logger.Errorf("Failed to apply service: %v", err)
				c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to apply service: %v", err)})
				return
			}
		case *networkingv1.Ingress:
			if err := a.applyIngress(o); err != nil {
				a.logger.Errorf("Failed to apply ingress: %v", err)
				c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to apply ingress: %v", err)})
				return
			}
		}
	}

	c.JSON(200, gin.H{"message": "Manifest applied successfully"})
}

func (a *API) applyDeployment(deployment *appsv1.Deployment) error {
	dep, err := a.parser.ParseDeployment(deployment)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", dep.Namespace, dep.Name)
	a.deployments[key] = dep

	// Create pods
	for i := int32(0); i < dep.DesiredReplicas; i++ {
		podName := fmt.Sprintf("%s-%d", dep.Name, i)
		pod := a.parser.ExtractPodFromTemplate(dep.Template, dep.Namespace, podName)

		// Generate pod ID
		pod.ID = generateID()
		pod.CreatedAt = time.Now().Unix()

		// Schedule pod
		nodeName, err := a.scheduler.SchedulePod(pod)
		if err != nil {
			return fmt.Errorf("failed to schedule pod: %w", err)
		}

		// Only create pod on this node if it's scheduled here
		if nodeName == a.cluster.GetLocalNodeName() {
			containerID, err := a.podman.CreatePod(pod)
			if err != nil {
				return fmt.Errorf("failed to create pod: %w", err)
			}

			pod.ID = containerID
			if err := a.podman.StartPod(containerID); err != nil {
				return fmt.Errorf("failed to start pod: %w", err)
			}

			// Update pod state
			state, _ := a.podman.GetPodStatus(containerID)
			a.scheduler.UpdatePodState(containerID, state)
		}

		dep.Pods = append(dep.Pods, pod)
	}

	a.logger.Infof("Applied deployment %s with %d replicas", key, dep.DesiredReplicas)
	return nil
}

func (a *API) applyService(service *corev1.Service) error {
	svc, err := a.parser.ParseService(service)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
	a.services[key] = svc

	// Register service in discovery for all matching pods
	pods := a.scheduler.GetAllPods()
	for _, pod := range pods {
		if matchesSelector(pod.Labels, svc.Selector) {
			if err := a.discovery.RegisterService(svc, pod); err != nil {
				a.logger.Warnf("Failed to register service for pod %s: %v", pod.Name, err)
			}
		}
	}

	a.logger.Infof("Applied service %s", key)
	return nil
}

func (a *API) applyIngress(ingress *networkingv1.Ingress) error {
	ing, err := a.parser.ParseIngress(ingress)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", ing.Namespace, ing.Name)
	a.ingresses[key] = ing

	if err := a.ingress.AddIngress(ing); err != nil {
		return err
	}

	a.logger.Infof("Applied ingress %s", key)
	return nil
}

func (a *API) DeleteManifest(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	key := fmt.Sprintf("%s/%s", namespace, name)

	// Try to delete deployment
	if dep, ok := a.deployments[key]; ok {
		for _, pod := range dep.Pods {
			if pod.NodeName == a.cluster.GetLocalNodeName() {
				if err := a.podman.StopPod(pod.ID); err != nil {
					a.logger.Warnf("Failed to stop pod %s: %v", pod.ID, err)
				}
				if err := a.podman.RemovePod(pod.ID); err != nil {
					a.logger.Warnf("Failed to remove pod %s: %v", pod.ID, err)
				}
			}
			a.scheduler.RemovePod(pod.ID)
		}
		delete(a.deployments, key)
	}

	// Try to delete service
	if svc, ok := a.services[key]; ok {
		pods := a.scheduler.GetAllPods()
		for _, pod := range pods {
			if matchesSelector(pod.Labels, svc.Selector) {
				a.discovery.DeregisterService(svc, pod)
			}
		}
		delete(a.services, key)
	}

	// Try to delete ingress
	if _, ok := a.ingresses[key]; ok {
		a.ingress.RemoveIngress(namespace, name)
		delete(a.ingresses, key)
	}

	c.JSON(200, gin.H{"message": "Manifest deleted successfully"})
}

func (a *API) ListPods(c *gin.Context) {
	pods := a.scheduler.GetAllPods()
	c.JSON(200, pods)
}

func (a *API) GetPod(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	pods := a.scheduler.GetAllPods()
	for _, pod := range pods {
		if pod.Namespace == namespace && pod.Name == name {
			c.JSON(200, pod)
			return
		}
	}

	c.JSON(404, gin.H{"error": "Pod not found"})
}

func (a *API) ListDeployments(c *gin.Context) {
	deployments := make([]*types.Deployment, 0, len(a.deployments))
	for _, dep := range a.deployments {
		deployments = append(deployments, dep)
	}
	c.JSON(200, deployments)
}

func (a *API) GetDeployment(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	key := fmt.Sprintf("%s/%s", namespace, name)
	if dep, ok := a.deployments[key]; ok {
		c.JSON(200, dep)
		return
	}

	c.JSON(404, gin.H{"error": "Deployment not found"})
}

func (a *API) ListServices(c *gin.Context) {
	services := make([]*types.Service, 0, len(a.services))
	for _, svc := range a.services {
		services = append(services, svc)
	}
	c.JSON(200, services)
}

func (a *API) ListNodes(c *gin.Context) {
	nodes := a.cluster.GetNodes()
	c.JSON(200, nodes)
}

func (a *API) GetServiceEndpoints(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	endpoints, err := a.discovery.GetServiceEndpoints(name, namespace)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, endpoints)
}

func (a *API) GetServiceAddresses(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	addresses, err := a.discovery.GetServiceAddresses(name, namespace)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"service":   name,
		"namespace": namespace,
		"addresses": addresses,
	})
}

func (a *API) Health(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"nodes":  a.cluster.GetNodeCount(),
	})
}

// DNS Whitelist endpoints

// GetDNSWhitelist returns the current DNS whitelist configuration
func (a *API) GetDNSWhitelist(c *gin.Context) {
	if a.dns == nil {
		c.JSON(500, gin.H{"error": "DNS server not available"})
		return
	}

	enabled, hosts := a.dns.GetWhitelist()
	c.JSON(200, gin.H{
		"enabled": enabled,
		"hosts":   hosts,
	})
}

// SetDNSWhitelist sets the DNS whitelist configuration
func (a *API) SetDNSWhitelist(c *gin.Context) {
	if a.dns == nil {
		c.JSON(500, gin.H{"error": "DNS server not available"})
		return
	}

	var req struct {
		Enabled bool     `json:"enabled"`
		Hosts   []string `json:"hosts"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	a.dns.SetWhitelist(req.Enabled, req.Hosts)
	a.logger.Infof("DNS whitelist updated: enabled=%v, hosts=%v", req.Enabled, req.Hosts)

	c.JSON(200, gin.H{
		"message": "DNS whitelist updated successfully",
		"enabled": req.Enabled,
		"hosts":   req.Hosts,
	})
}

// AddDNSWhitelistHost adds a host to the DNS whitelist
func (a *API) AddDNSWhitelistHost(c *gin.Context) {
	if a.dns == nil {
		c.JSON(500, gin.H{"error": "DNS server not available"})
		return
	}

	var req struct {
		Host string `json:"host"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	if req.Host == "" {
		c.JSON(400, gin.H{"error": "Host is required"})
		return
	}

	enabled, hosts := a.dns.GetWhitelist()
	// Add new host if not already present
	found := false
	for _, h := range hosts {
		if h == req.Host {
			found = true
			break
		}
	}
	if !found {
		hosts = append(hosts, req.Host)
	}
	a.dns.SetWhitelist(enabled, hosts)

	a.logger.Infof("Added host to DNS whitelist: %s", req.Host)
	c.JSON(200, gin.H{
		"message": "Host added to whitelist",
		"host":    req.Host,
	})
}

// RemoveDNSWhitelistHost removes a host from the DNS whitelist
func (a *API) RemoveDNSWhitelistHost(c *gin.Context) {
	if a.dns == nil {
		c.JSON(500, gin.H{"error": "DNS server not available"})
		return
	}

	host := c.Param("host")
	if host == "" {
		c.JSON(400, gin.H{"error": "Host parameter is required"})
		return
	}

	enabled, hosts := a.dns.GetWhitelist()
	// Remove host from list
	newHosts := make([]string, 0, len(hosts))
	for _, h := range hosts {
		if h != host {
			newHosts = append(newHosts, h)
		}
	}
	a.dns.SetWhitelist(enabled, newHosts)

	a.logger.Infof("Removed host from DNS whitelist: %s", host)
	c.JSON(200, gin.H{
		"message": "Host removed from whitelist",
		"host":    host,
	})
}

// API Token management endpoints

// GenerateAPIToken generates a new API token
func (a *API) GenerateAPIToken(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		ExpiresIn int    `json:"expires_in"` // Seconds, 0 means no expiration
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	if req.Name == "" {
		c.JSON(400, gin.H{"error": "Token name is required"})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		expiresAt = &expiry
	}

	token, err := a.tokenManager.GenerateToken(req.Name, expiresAt)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate token: %v", err)})
		return
	}

	a.logger.Infof("Generated new API token: %s (expires: %v)", req.Name, expiresAt)

	c.JSON(201, gin.H{
		"message":    "Token generated successfully",
		"token":      token,
		"name":       req.Name,
		"expires_at": expiresAt,
	})
}

// ListAPITokens lists all API tokens (without showing actual token values)
func (a *API) ListAPITokens(c *gin.Context) {
	tokens := a.tokenManager.ListTokens()
	c.JSON(200, gin.H{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

// RevokeAPIToken revokes an API token
func (a *API) RevokeAPIToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(400, gin.H{"error": "Token parameter is required"})
		return
	}

	if err := a.tokenManager.RevokeToken(token); err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("Token not found: %v", err)})
		return
	}

	a.logger.Infof("Revoked API token: %s", token)
	c.JSON(200, gin.H{
		"message": "Token revoked successfully",
	})
}

// Helper functions

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func matchesSelector(labels, selector map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}
