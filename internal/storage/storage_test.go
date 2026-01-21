package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/your-server-support/podman-swarm/internal/types"
)

func setupTestStorage(t *testing.T) (*Storage, string) {
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs in tests

	storage, err := NewStorage(StorageConfig{
		DataDir: tmpDir,
		Logger:  logger,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	return storage, tmpDir
}

func cleanup(tmpDir string) {
	os.RemoveAll(tmpDir)
}

func TestNewStorage(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	if storage == nil {
		t.Fatal("Storage should not be nil")
	}

	// Check data directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Errorf("Data directory was not created: %s", tmpDir)
	}
}

func TestSaveAndGetDeployment(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	deployment := &types.Deployment{
		Name:            "test-deployment",
		Namespace:       "default",
		DesiredReplicas: 3,
		Pods:            []*types.Pod{},
	}

	// Save deployment
	err := storage.SaveDeployment(deployment)
	if err != nil {
		t.Errorf("Failed to save deployment: %v", err)
	}

	// Get deployment
	retrieved, err := storage.GetDeployment("default", "test-deployment")
	if err != nil {
		t.Errorf("Failed to get deployment: %v", err)
	}

	if retrieved.Name != deployment.Name {
		t.Errorf("Expected name %s, got %s", deployment.Name, retrieved.Name)
	}

	if retrieved.Namespace != deployment.Namespace {
		t.Errorf("Expected namespace %s, got %s", deployment.Namespace, retrieved.Namespace)
	}

	if retrieved.DesiredReplicas != deployment.DesiredReplicas {
		t.Errorf("Expected %d replicas, got %d", deployment.DesiredReplicas, retrieved.DesiredReplicas)
	}
}

func TestDeleteDeployment(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	deployment := &types.Deployment{
		Name:      "test-deployment",
		Namespace: "default",
	}

	// Save and then delete
	storage.SaveDeployment(deployment)
	err := storage.DeleteDeployment("default", "test-deployment")
	if err != nil {
		t.Errorf("Failed to delete deployment: %v", err)
	}

	// Verify it's gone
	_, err = storage.GetDeployment("default", "test-deployment")
	if err == nil {
		t.Error("Expected error when getting deleted deployment")
	}
}

func TestListDeployments(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	// Save multiple deployments
	for i := 1; i <= 3; i++ {
		deployment := &types.Deployment{
			Name:      "test-" + string(rune('0'+i)),
			Namespace: "default",
		}
		storage.SaveDeployment(deployment)
	}

	deployments := storage.ListDeployments()
	if len(deployments) != 3 {
		t.Errorf("Expected 3 deployments, got %d", len(deployments))
	}
}

func TestSaveAndGetService(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	service := &types.Service{
		Name:      "test-service",
		Namespace: "default",
		Selector:  map[string]string{"app": "test"},
	}

	err := storage.SaveService(service)
	if err != nil {
		t.Errorf("Failed to save service: %v", err)
	}

	retrieved, err := storage.GetService("default", "test-service")
	if err != nil {
		t.Errorf("Failed to get service: %v", err)
	}

	if retrieved.Name != service.Name {
		t.Errorf("Expected name %s, got %s", service.Name, retrieved.Name)
	}
}

func TestPersistence(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)

	deployment := &types.Deployment{
		Name:      "persistent-test",
		Namespace: "default",
	}

	storage.SaveDeployment(deployment)

	// Check state file exists
	stateFile := filepath.Join(tmpDir, "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file was not created")
	}

	// Create new storage instance with same directory
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	newStorage, err := NewStorage(StorageConfig{
		DataDir: tmpDir,
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("Failed to create new storage: %v", err)
	}

	// Verify data was loaded
	retrieved, err := newStorage.GetDeployment("default", "persistent-test")
	if err != nil {
		t.Errorf("Failed to get deployment from new storage: %v", err)
	}

	if retrieved.Name != deployment.Name {
		t.Errorf("Expected name %s, got %s", deployment.Name, retrieved.Name)
	}

	cleanup(tmpDir)
}

func TestBackup(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	deployment := &types.Deployment{
		Name:      "backup-test",
		Namespace: "default",
	}

	storage.SaveDeployment(deployment)

	err := storage.Backup()
	if err != nil {
		t.Errorf("Failed to create backup: %v", err)
	}

	// Check backup file exists
	files, err := filepath.Glob(filepath.Join(tmpDir, "state-backup-*.json"))
	if err != nil {
		t.Errorf("Failed to glob backup files: %v", err)
	}

	if len(files) == 0 {
		t.Error("No backup file was created")
	}
}

func TestMergeState(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	// Create local state
	localDeployment := &types.Deployment{
		Name:      "local",
		Namespace: "default",
	}
	storage.SaveDeployment(localDeployment)

	// Create incoming state (newer)
	time.Sleep(10 * time.Millisecond) // Ensure newer timestamp
	incomingState := &ClusterState{
		Deployments: map[string]*types.Deployment{
			"default/remote": {
				Name:      "remote",
				Namespace: "default",
			},
		},
		Services:     make(map[string]*types.Service),
		Ingresses:    make(map[string]*types.Ingress),
		Pods:         make(map[string]*types.Pod),
		LastModified: time.Now(),
		Version:      1,
	}

	err := storage.MergeState(incomingState)
	if err != nil {
		t.Errorf("Failed to merge state: %v", err)
	}

	// Verify both deployments exist
	deployments := storage.ListDeployments()
	if len(deployments) != 2 {
		t.Errorf("Expected 2 deployments after merge, got %d", len(deployments))
	}
}

func TestGetState(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	deployment := &types.Deployment{
		Name:      "test",
		Namespace: "default",
	}
	storage.SaveDeployment(deployment)

	state := storage.GetState()
	if state == nil {
		t.Fatal("GetState returned nil")
	}

	if len(state.Deployments) != 1 {
		t.Errorf("Expected 1 deployment in state, got %d", len(state.Deployments))
	}

	if state.Version != 1 {
		t.Errorf("Expected version 1, got %d", state.Version)
	}
}

func TestConcurrentAccess(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			deployment := &types.Deployment{
				Name:      "concurrent-" + string(rune('0'+idx)),
				Namespace: "default",
			}
			storage.SaveDeployment(deployment)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all deployments were saved
	deployments := storage.ListDeployments()
	if len(deployments) != 10 {
		t.Errorf("Expected 10 deployments, got %d", len(deployments))
	}
}

func TestAtomicWrites(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	deployment := &types.Deployment{
		Name:      "atomic-test",
		Namespace: "default",
	}

	storage.SaveDeployment(deployment)

	// Verify no .tmp file exists (should be cleaned up)
	tmpFiles, _ := filepath.Glob(filepath.Join(tmpDir, "*.tmp"))
	if len(tmpFiles) > 0 {
		t.Error("Temporary files were not cleaned up")
	}

	// Verify state.json exists
	stateFile := filepath.Join(tmpDir, "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file does not exist")
	}
}

func TestSaveIngress(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	ingress := &types.Ingress{
		Name:      "test-ingress",
		Namespace: "default",
		Rules: []types.IngressRule{
			{
				Host: "example.com",
			},
		},
	}

	err := storage.SaveIngress(ingress)
	if err != nil {
		t.Errorf("Failed to save ingress: %v", err)
	}

	retrieved, err := storage.GetIngress("default", "test-ingress")
	if err != nil {
		t.Errorf("Failed to get ingress: %v", err)
	}

	if retrieved.Name != ingress.Name {
		t.Errorf("Expected name %s, got %s", ingress.Name, retrieved.Name)
	}

	if len(retrieved.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(retrieved.Rules))
	}
}

func TestSavePod(t *testing.T) {
	storage, tmpDir := setupTestStorage(t)
	defer cleanup(tmpDir)

	pod := &types.Pod{
		Name:      "test-pod",
		Namespace: "default",
		NodeName:  "node-1",
		State:     types.PodStateRunning,
	}

	err := storage.SavePod(pod)
	if err != nil {
		t.Errorf("Failed to save pod: %v", err)
	}

	retrieved, err := storage.GetPod("default", "test-pod")
	if err != nil {
		t.Errorf("Failed to get pod: %v", err)
	}

	if retrieved.Name != pod.Name {
		t.Errorf("Expected name %s, got %s", pod.Name, retrieved.Name)
	}

	if retrieved.State != types.PodStateRunning {
		t.Errorf("Expected state %v, got %v", types.PodStateRunning, retrieved.State)
	}
}
