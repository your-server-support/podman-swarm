package api

import (
	"fmt"
	"time"

	"github.com/your-server-support/podman-swarm/internal/types"
)

// RecoverDeployments recovers deployments from persistent storage
// This is called during agent startup to restore cluster state
func (a *API) RecoverDeployments() error {
	a.logger.Info("Starting deployment recovery from persistent storage...")

	deployments := a.storage.ListDeployments()
	recoveredCount := 0
	failedCount := 0

	for _, dep := range deployments {
		a.logger.Infof("Recovering deployment: %s/%s", dep.Namespace, dep.Name)

		// Clear existing pods (they will be recreated)
		dep.Pods = []*types.Pod{}

		// Recreate pods according to desired replicas
		for i := int32(0); i < dep.DesiredReplicas; i++ {
			podName := fmt.Sprintf("%s-%d", dep.Name, i)
			pod := a.parser.ExtractPodFromTemplate(dep.Template, dep.Namespace, podName)

			// Generate pod ID
			pod.ID = generateID()
			pod.CreatedAt = time.Now().Unix()

			// Schedule pod
			nodeName, err := a.scheduler.SchedulePod(pod)
			if err != nil {
				a.logger.Errorf("Failed to schedule pod %s during recovery: %v", podName, err)
				failedCount++
				continue
			}

			// Only create pod on this node if it's scheduled here
			if nodeName == a.cluster.GetLocalNodeName() {
				containerID, err := a.podman.CreatePod(pod)
				if err != nil {
					a.logger.Errorf("Failed to create pod %s during recovery: %v", podName, err)
					failedCount++
					continue
				}

				pod.ID = containerID
				if err := a.podman.StartPod(containerID); err != nil {
					a.logger.Errorf("Failed to start pod %s during recovery: %v", podName, err)
					failedCount++
					continue
				}

				// Update pod state
				state, _ := a.podman.GetPodStatus(containerID)
				a.scheduler.UpdatePodState(containerID, state)

				a.logger.Infof("Recovered pod %s on node %s", podName, nodeName)
			}

			dep.Pods = append(dep.Pods, pod)
		}

		// Update deployment in storage
		key := fmt.Sprintf("%s/%s", dep.Namespace, dep.Name)
		a.deployments[key] = dep
		if err := a.storage.SaveDeployment(dep); err != nil {
			a.logger.Warnf("Failed to update deployment after recovery: %v", err)
		}

		recoveredCount++
	}

	// Recover services
	services := a.storage.ListServices()
	for _, svc := range services {
		a.logger.Infof("Recovering service: %s/%s", svc.Namespace, svc.Name)

		// Register service in discovery for all matching pods
		pods := a.scheduler.GetAllPods()
		for _, pod := range pods {
			if matchesSelector(pod.Labels, svc.Selector) {
				if err := a.discovery.RegisterService(svc, pod); err != nil {
					a.logger.Warnf("Failed to register service for pod %s during recovery: %v", pod.Name, err)
				}
			}
		}

		key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
		a.services[key] = svc
	}

	a.logger.Infof("Deployment recovery completed: %d recovered, %d failed", recoveredCount, failedCount)

	if failedCount > 0 {
		return fmt.Errorf("%d pods failed to recover", failedCount)
	}

	return nil
}

// StartStateRecovery starts the state recovery process
// Should be called after API initialization
func (a *API) StartStateRecovery() {
	go func() {
		// Wait a bit for cluster to stabilize
		time.Sleep(5 * time.Second)

		if err := a.RecoverDeployments(); err != nil {
			a.logger.Errorf("Deployment recovery encountered errors: %v", err)
		}
	}()
}
