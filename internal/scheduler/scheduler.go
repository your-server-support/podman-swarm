package scheduler

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/your-server-support/podman-swarm/internal/cluster"
	"github.com/your-server-support/podman-swarm/internal/types"
)

type Scheduler struct {
	cluster *cluster.Cluster
	logger  *logrus.Logger
	mu      sync.RWMutex
	pods    map[string]*types.Pod // podID -> pod
}

func NewScheduler(cluster *cluster.Cluster, logger *logrus.Logger) *Scheduler {
	return &Scheduler{
		cluster: cluster,
		logger:  logger,
		pods:    make(map[string]*types.Pod),
	}
}

// SchedulePod schedules a pod to a node
func (s *Scheduler) SchedulePod(pod *types.Pod) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check node selector
	if len(pod.NodeSelector) > 0 {
		node, err := s.findNodeBySelector(pod.NodeSelector)
		if err != nil {
			return "", fmt.Errorf("no node matches selector: %w", err)
		}
		pod.NodeName = node.Name
		s.pods[pod.ID] = pod
		return node.Name, nil
	}

	// Select a random node (simple round-robin or random)
	nodes := s.cluster.GetNodes()
	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes available")
	}

	// Simple random selection (can be improved with resource-based scheduling)
	rand.Seed(time.Now().UnixNano())
	selectedNode := nodes[rand.Intn(len(nodes))]
	pod.NodeName = selectedNode.Name

	s.pods[pod.ID] = pod
	s.logger.Infof("Scheduled pod %s to node %s", pod.Name, selectedNode.Name)

	return selectedNode.Name, nil
}

// findNodeBySelector finds a node that matches the selector
func (s *Scheduler) findNodeBySelector(selector map[string]string) (*types.Node, error) {
	nodes := s.cluster.GetNodes()

	for _, node := range nodes {
		matches := true
		for key, value := range selector {
			if node.Labels[key] != value {
				matches = false
				break
			}
		}
		if matches {
			return node, nil
		}
	}

	return nil, fmt.Errorf("no node matches selector")
}

// GetPod returns a pod by ID
func (s *Scheduler) GetPod(podID string) (*types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pod, ok := s.pods[podID]
	if !ok {
		return nil, fmt.Errorf("pod %s not found", podID)
	}

	return pod, nil
}

// GetPodsByNode returns all pods scheduled on a node
func (s *Scheduler) GetPodsByNode(nodeName string) []*types.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*types.Pod
	for _, pod := range s.pods {
		if pod.NodeName == nodeName {
			result = append(result, pod)
		}
	}

	return result
}

// RemovePod removes a pod from the scheduler
func (s *Scheduler) RemovePod(podID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pods, podID)
}

// GetAllPods returns all pods
func (s *Scheduler) GetAllPods() []*types.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pods := make([]*types.Pod, 0, len(s.pods))
	for _, pod := range s.pods {
		pods = append(pods, pod)
	}

	return pods
}

// UpdatePodState updates the state of a pod
func (s *Scheduler) UpdatePodState(podID string, state types.PodState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pod, ok := s.pods[podID]
	if !ok {
		return fmt.Errorf("pod %s not found", podID)
	}

	pod.State = state
	return nil
}
