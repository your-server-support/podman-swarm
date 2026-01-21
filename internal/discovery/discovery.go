package discovery

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/your-server-support/podman-swarm/internal/cluster"
	"github.com/your-server-support/podman-swarm/internal/types"
)

// ServiceEndpoint represents a service endpoint
type ServiceEndpoint struct {
	ServiceName string
	Namespace   string
	PodID       string
	PodName     string
	NodeName    string
	Address     string
	Port        int32
	Healthy     bool
	LastSeen    time.Time
}

// ServiceRegistry stores service information
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]map[string]*ServiceEndpoint // serviceKey -> endpointID -> endpoint
	logger   *logrus.Logger
	cluster  *cluster.Cluster
}

type Discovery struct {
	registry *ServiceRegistry
	logger   *logrus.Logger
	cluster  *cluster.Cluster
}

func NewDiscovery(cluster *cluster.Cluster, logger *logrus.Logger) *Discovery {
	registry := &ServiceRegistry{
		services: make(map[string]map[string]*ServiceEndpoint),
		logger:   logger,
		cluster:  cluster,
	}

	discovery := &Discovery{
		registry: registry,
		logger:   logger,
		cluster:  cluster,
	}

	// Start health check routine
	go discovery.healthCheck()

	return discovery
}

// serviceKey generates a key for a service
func serviceKey(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s", serviceName, namespace)
}

// endpointID generates an ID for an endpoint
func endpointID(namespace, serviceName, podID string) string {
	return fmt.Sprintf("%s-%s-%s", namespace, serviceName, podID)
}

// RegisterService registers a service endpoint
func (d *Discovery) RegisterService(service *types.Service, pod *types.Pod) error {
	key := serviceKey(service.Name, service.Namespace)
	endpointID := endpointID(service.Namespace, service.Name, pod.ID)

	var port int32
	if len(service.Ports) > 0 {
		port = service.Ports[0].Port
	}

	// Get node address from cluster
	nodeAddress := pod.NodeName // Default to node name
	if node, err := d.cluster.GetNode(pod.NodeName); err == nil {
		nodeAddress = node.Address
	}

	endpoint := &ServiceEndpoint{
		ServiceName: service.Name,
		Namespace:   service.Namespace,
		PodID:       pod.ID,
		PodName:     pod.Name,
		NodeName:    pod.NodeName,
		Address:     nodeAddress, // Use actual node address
		Port:        port,
		Healthy:     true,
		LastSeen:    time.Now(),
	}

	d.registry.mu.Lock()
	defer d.registry.mu.Unlock()

	if d.registry.services[key] == nil {
		d.registry.services[key] = make(map[string]*ServiceEndpoint)
	}

	d.registry.services[key][endpointID] = endpoint

	// Broadcast service registration to cluster
	d.broadcastServiceUpdate(endpoint, "register")

	d.logger.Infof("Registered service %s for pod %s on node %s", service.Name, pod.Name, pod.NodeName)
	return nil
}

// DeregisterService removes a service endpoint
func (d *Discovery) DeregisterService(service *types.Service, pod *types.Pod) error {
	key := serviceKey(service.Name, service.Namespace)
	endpointID := endpointID(service.Namespace, service.Name, pod.ID)

	d.registry.mu.Lock()
	defer d.registry.mu.Unlock()

	if endpoints, ok := d.registry.services[key]; ok {
		if endpoint, exists := endpoints[endpointID]; exists {
			// Broadcast service deregistration
			d.broadcastServiceUpdate(endpoint, "deregister")
			delete(endpoints, endpointID)

			if len(endpoints) == 0 {
				delete(d.registry.services, key)
			}

			d.logger.Infof("Deregistered service %s for pod %s", service.Name, pod.Name)
		}
	}

	return nil
}

// GetServiceAddresses returns addresses of healthy service instances
func (d *Discovery) GetServiceAddresses(serviceName, namespace string) ([]string, error) {
	key := serviceKey(serviceName, namespace)

	d.registry.mu.RLock()
	defer d.registry.mu.RUnlock()

	endpoints, ok := d.registry.services[key]
	if !ok {
		return nil, fmt.Errorf("service %s not found", key)
	}

	addresses := make([]string, 0)
	for _, endpoint := range endpoints {
		if endpoint.Healthy {
			// Check if endpoint is still fresh (within last 30 seconds)
			if time.Since(endpoint.LastSeen) < 30*time.Second {
				addresses = append(addresses, fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port))
			}
		}
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("no healthy instances for service %s", key)
	}

	return addresses, nil
}

// GetServiceEndpoints returns all endpoints for a service
func (d *Discovery) GetServiceEndpoints(serviceName, namespace string) ([]*ServiceEndpoint, error) {
	key := serviceKey(serviceName, namespace)

	d.registry.mu.RLock()
	defer d.registry.mu.RUnlock()

	endpoints, ok := d.registry.services[key]
	if !ok {
		return nil, fmt.Errorf("service %s not found", key)
	}

	result := make([]*ServiceEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.Healthy && time.Since(endpoint.LastSeen) < 30*time.Second {
			result = append(result, endpoint)
		}
	}

	return result, nil
}

// broadcastServiceUpdate broadcasts service update to cluster
func (d *Discovery) broadcastServiceUpdate(endpoint *ServiceEndpoint, action string) {
	message := map[string]interface{}{
		"type":        "service_update",
		"action":      action,
		"serviceName": endpoint.ServiceName,
		"namespace":   endpoint.Namespace,
		"podID":       endpoint.PodID,
		"podName":     endpoint.PodName,
		"nodeName":    endpoint.NodeName,
		"address":     endpoint.Address,
		"port":        endpoint.Port,
		"healthy":     endpoint.Healthy,
		"timestamp":   time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		d.logger.Errorf("Failed to marshal service update: %v", err)
		return
	}

	if err := d.cluster.Broadcast(data); err != nil {
		d.logger.Warnf("Failed to broadcast service update: %v", err)
	}
}

// HandleServiceUpdate handles service update from cluster
func (d *Discovery) HandleServiceUpdate(message []byte) error {
	var update map[string]interface{}
	if err := json.Unmarshal(message, &update); err != nil {
		return fmt.Errorf("failed to unmarshal service update: %w", err)
	}

	if update["type"] != "service_update" {
		return nil // Not a service update
	}

	action := update["action"].(string)
	key := serviceKey(update["serviceName"].(string), update["namespace"].(string))
	endpointID := endpointID(update["namespace"].(string), update["serviceName"].(string), update["podID"].(string))

	endpoint := &ServiceEndpoint{
		ServiceName: update["serviceName"].(string),
		Namespace:   update["namespace"].(string),
		PodID:       update["podID"].(string),
		PodName:     update["podName"].(string),
		NodeName:    update["nodeName"].(string),
		Address:     update["address"].(string),
		Port:        int32(update["port"].(float64)),
		Healthy:     update["healthy"].(bool),
		LastSeen:    time.Now(),
	}

	d.registry.mu.Lock()
	defer d.registry.mu.Unlock()

	switch action {
	case "register":
		if d.registry.services[key] == nil {
			d.registry.services[key] = make(map[string]*ServiceEndpoint)
		}
		d.registry.services[key][endpointID] = endpoint
		d.logger.Debugf("Received service registration: %s", key)

	case "deregister":
		if endpoints, ok := d.registry.services[key]; ok {
			delete(endpoints, endpointID)
			if len(endpoints) == 0 {
				delete(d.registry.services, key)
			}
			d.logger.Debugf("Received service deregistration: %s", key)
		}
	}

	return nil
}

// healthCheck periodically checks service health
func (d *Discovery) healthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		d.registry.mu.Lock()
		now := time.Now()
		for _, endpoints := range d.registry.services {
			for endpointID, endpoint := range endpoints {
				// Mark as unhealthy if not seen for 30 seconds
				if now.Sub(endpoint.LastSeen) > 30*time.Second {
					endpoint.Healthy = false
					d.logger.Debugf("Marked endpoint %s as unhealthy", endpointID)
				}
			}
		}
		d.registry.mu.Unlock()
	}
}

// ListServices returns all registered services
func (d *Discovery) ListServices() map[string][]*ServiceEndpoint {
	d.registry.mu.RLock()
	defer d.registry.mu.RUnlock()

	result := make(map[string][]*ServiceEndpoint)
	for key, endpoints := range d.registry.services {
		healthyEndpoints := make([]*ServiceEndpoint, 0)
		for _, endpoint := range endpoints {
			if endpoint.Healthy && time.Since(endpoint.LastSeen) < 30*time.Second {
				healthyEndpoints = append(healthyEndpoints, endpoint)
			}
		}
		if len(healthyEndpoints) > 0 {
			result[key] = healthyEndpoints
		}
	}

	return result
}
