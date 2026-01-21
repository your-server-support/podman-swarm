package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// ServiceClient is a client for discovering and connecting to services
type ServiceClient struct {
	apiBaseURL string
	httpClient *http.Client
}

// NewServiceClient creates a new service client
func NewServiceClient(apiBaseURL string) *ServiceClient {
	return &ServiceClient{
		apiBaseURL: apiBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// ServiceEndpoint represents a service endpoint
type ServiceEndpoint struct {
	ServiceName string    `json:"service_name"`
	Namespace   string    `json:"namespace"`
	PodID       string    `json:"pod_id"`
	PodName     string    `json:"pod_name"`
	NodeName    string    `json:"node_name"`
	Address     string    `json:"address"`
	Port        int32     `json:"port"`
	Healthy     bool      `json:"healthy"`
	LastSeen    time.Time `json:"last_seen"`
}

// GetServiceEndpoints returns all endpoints for a service
func (sc *ServiceClient) GetServiceEndpoints(namespace, serviceName string) ([]ServiceEndpoint, error) {
	url := fmt.Sprintf("%s/api/v1/services/%s/%s/endpoints", sc.apiBaseURL, namespace, serviceName)
	
	resp, err := sc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var endpoints []ServiceEndpoint
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return endpoints, nil
}

// GetServiceAddresses returns addresses of service instances
func (sc *ServiceClient) GetServiceAddresses(namespace, serviceName string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/services/%s/%s/addresses", sc.apiBaseURL, namespace, serviceName)
	
	resp, err := sc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Service   string   `json:"service"`
		Namespace string   `json:"namespace"`
		Addresses []string `json:"addresses"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Addresses, nil
}

// ConnectToService establishes a TCP connection to a service
func (sc *ServiceClient) ConnectToService(namespace, serviceName string) (net.Conn, error) {
	// Get service addresses
	addresses, err := sc.GetServiceAddresses(namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("no healthy endpoints found for service %s", serviceName)
	}

	// Simple round-robin: use first address
	// In production, you might want to implement more sophisticated selection
	target := addresses[0]

	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
	}

	return conn, nil
}

// Example usage
func main() {
	// Create service client
	client := NewServiceClient("http://localhost:8080")

	// Example 1: Get all endpoints
	endpoints, err := client.GetServiceEndpoints("default", "postgres-service")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d endpoints:\n", len(endpoints))
	for _, ep := range endpoints {
		fmt.Printf("  - %s:%d on node %s (healthy: %v)\n", 
			ep.Address, ep.Port, ep.NodeName, ep.Healthy)
	}

	// Example 2: Get addresses
	addresses, err := client.GetServiceAddresses("default", "postgres-service")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nService addresses: %v\n", addresses)

	// Example 3: Connect to service
	conn, err := client.ConnectToService("default", "postgres-service")
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("\nConnected to service at %s\n", conn.RemoteAddr().String())
}
