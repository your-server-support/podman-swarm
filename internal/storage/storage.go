package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/your-server-support/podman-swarm/internal/types"
)

// Storage represents persistent storage for cluster state
type Storage struct {
	dataDir      string
	logger       *logrus.Logger
	mu           sync.RWMutex
	deployments  map[string]*types.Deployment
	services     map[string]*types.Service
	ingresses    map[string]*types.Ingress
	pods         map[string]*types.Pod
	lastModified time.Time
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	DataDir string
	Logger  *logrus.Logger
}

// NewStorage creates a new storage instance
func NewStorage(config StorageConfig) (*Storage, error) {
	if config.DataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	s := &Storage{
		dataDir:     config.DataDir,
		logger:      config.Logger,
		deployments: make(map[string]*types.Deployment),
		services:    make(map[string]*types.Service),
		ingresses:   make(map[string]*types.Ingress),
		pods:        make(map[string]*types.Pod),
	}

	// Load existing state
	if err := s.Load(); err != nil {
		s.logger.Warnf("Failed to load existing state: %v", err)
		// Continue anyway - start with empty state
	}

	return s, nil
}

// SaveDeployment saves a deployment to persistent storage
func (s *Storage) SaveDeployment(deployment *types.Deployment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	s.deployments[key] = deployment
	s.lastModified = time.Now()

	return s.persist()
}

// GetDeployment retrieves a deployment from storage
func (s *Storage) GetDeployment(namespace, name string) (*types.Deployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	deployment, ok := s.deployments[key]
	if !ok {
		return nil, fmt.Errorf("deployment not found: %s/%s", namespace, name)
	}

	return deployment, nil
}

// DeleteDeployment removes a deployment from storage
func (s *Storage) DeleteDeployment(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	delete(s.deployments, key)
	s.lastModified = time.Now()

	return s.persist()
}

// ListDeployments returns all deployments
func (s *Storage) ListDeployments() []*types.Deployment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deployments := make([]*types.Deployment, 0, len(s.deployments))
	for _, d := range s.deployments {
		deployments = append(deployments, d)
	}

	return deployments
}

// SaveService saves a service to persistent storage
func (s *Storage) SaveService(service *types.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	s.services[key] = service
	s.lastModified = time.Now()

	return s.persist()
}

// GetService retrieves a service from storage
func (s *Storage) GetService(namespace, name string) (*types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	service, ok := s.services[key]
	if !ok {
		return nil, fmt.Errorf("service not found: %s/%s", namespace, name)
	}

	return service, nil
}

// DeleteService removes a service from storage
func (s *Storage) DeleteService(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	delete(s.services, key)
	s.lastModified = time.Now()

	return s.persist()
}

// ListServices returns all services
func (s *Storage) ListServices() []*types.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make([]*types.Service, 0, len(s.services))
	for _, svc := range s.services {
		services = append(services, svc)
	}

	return services
}

// SaveIngress saves an ingress to persistent storage
func (s *Storage) SaveIngress(ingress *types.Ingress) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", ingress.Namespace, ingress.Name)
	s.ingresses[key] = ingress
	s.lastModified = time.Now()

	return s.persist()
}

// GetIngress retrieves an ingress from storage
func (s *Storage) GetIngress(namespace, name string) (*types.Ingress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	ingress, ok := s.ingresses[key]
	if !ok {
		return nil, fmt.Errorf("ingress not found: %s/%s", namespace, name)
	}

	return ingress, nil
}

// DeleteIngress removes an ingress from storage
func (s *Storage) DeleteIngress(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	delete(s.ingresses, key)
	s.lastModified = time.Now()

	return s.persist()
}

// ListIngresses returns all ingresses
func (s *Storage) ListIngresses() []*types.Ingress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ingresses := make([]*types.Ingress, 0, len(s.ingresses))
	for _, ing := range s.ingresses {
		ingresses = append(ingresses, ing)
	}

	return ingresses
}

// SavePod saves a pod to persistent storage
func (s *Storage) SavePod(pod *types.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	s.pods[key] = pod
	s.lastModified = time.Now()

	return s.persist()
}

// GetPod retrieves a pod from storage
func (s *Storage) GetPod(namespace, name string) (*types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	pod, ok := s.pods[key]
	if !ok {
		return nil, fmt.Errorf("pod not found: %s/%s", namespace, name)
	}

	return pod, nil
}

// DeletePod removes a pod from storage
func (s *Storage) DeletePod(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	delete(s.pods, key)
	s.lastModified = time.Now()

	return s.persist()
}

// ListPods returns all pods
func (s *Storage) ListPods() []*types.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pods := make([]*types.Pod, 0, len(s.pods))
	for _, pod := range s.pods {
		pods = append(pods, pod)
	}

	return pods
}

// ClusterState represents the complete cluster state
type ClusterState struct {
	Deployments  map[string]*types.Deployment `json:"deployments"`
	Services     map[string]*types.Service    `json:"services"`
	Ingresses    map[string]*types.Ingress    `json:"ingresses"`
	Pods         map[string]*types.Pod        `json:"pods"`
	LastModified time.Time                    `json:"last_modified"`
	Version      int                          `json:"version"`
}

// persist writes the current state to disk
func (s *Storage) persist() error {
	state := ClusterState{
		Deployments:  s.deployments,
		Services:     s.services,
		Ingresses:    s.ingresses,
		Pods:         s.pods,
		LastModified: s.lastModified,
		Version:      1,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first
	tmpFile := filepath.Join(s.dataDir, "state.json.tmp")
	if err := os.WriteFile(tmpFile, data, 0640); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	// Atomic rename
	stateFile := filepath.Join(s.dataDir, "state.json")
	if err := os.Rename(tmpFile, stateFile); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	s.logger.Debugf("State persisted to %s", stateFile)
	return nil
}

// Load reads the state from disk
func (s *Storage) Load() error {
	stateFile := filepath.Join(s.dataDir, "state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Info("No existing state file found, starting fresh")
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state ClusterState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.deployments = state.Deployments
	if s.deployments == nil {
		s.deployments = make(map[string]*types.Deployment)
	}

	s.services = state.Services
	if s.services == nil {
		s.services = make(map[string]*types.Service)
	}

	s.ingresses = state.Ingresses
	if s.ingresses == nil {
		s.ingresses = make(map[string]*types.Ingress)
	}

	s.pods = state.Pods
	if s.pods == nil {
		s.pods = make(map[string]*types.Pod)
	}

	s.lastModified = state.LastModified

	s.logger.Infof("Loaded state: %d deployments, %d services, %d ingresses, %d pods",
		len(s.deployments), len(s.services), len(s.ingresses), len(s.pods))

	return nil
}

// GetState returns the current cluster state for synchronization
func (s *Storage) GetState() *ClusterState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &ClusterState{
		Deployments:  s.deployments,
		Services:     s.services,
		Ingresses:    s.ingresses,
		Pods:         s.pods,
		LastModified: s.lastModified,
		Version:      1,
	}
}

// MergeState merges incoming state with local state
// Used for peer-to-peer synchronization
func (s *Storage) MergeState(incomingState *ClusterState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Simple merge strategy: use incoming state if it's newer
	if incomingState.LastModified.After(s.lastModified) {
		s.logger.Infof("Merging newer state from peer (incoming: %s, local: %s)",
			incomingState.LastModified, s.lastModified)

		// Merge deployments
		for key, deployment := range incomingState.Deployments {
			s.deployments[key] = deployment
		}

		// Merge services
		for key, service := range incomingState.Services {
			s.services[key] = service
		}

		// Merge ingresses
		for key, ingress := range incomingState.Ingresses {
			s.ingresses[key] = ingress
		}

		// Note: Pods are typically node-specific, so we might want different logic here
		// For now, we'll merge them as well
		for key, pod := range incomingState.Pods {
			if existing, ok := s.pods[key]; !ok || pod.CreatedAt > existing.CreatedAt {
				s.pods[key] = pod
			}
		}

		s.lastModified = time.Now()
		return s.persist()
	}

	return nil
}

// Backup creates a backup of the current state
func (s *Storage) Backup() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(s.dataDir, fmt.Sprintf("state-backup-%s.json", timestamp))

	state := ClusterState{
		Deployments:  s.deployments,
		Services:     s.services,
		Ingresses:    s.ingresses,
		Pods:         s.pods,
		LastModified: s.lastModified,
		Version:      1,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state for backup: %w", err)
	}

	if err := os.WriteFile(backupFile, data, 0640); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	s.logger.Infof("State backup created: %s", backupFile)
	return nil
}
