package types

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodState represents the state of a pod
type PodState string

const (
	PodStatePending   PodState = "Pending"
	PodStateRunning   PodState = "Running"
	PodStateSucceeded PodState = "Succeeded"
	PodStateFailed    PodState = "Failed"
	PodStateUnknown   PodState = "Unknown"
)

// Pod represents a pod in the cluster
type Pod struct {
	ID          string
	Name        string
	Namespace   string
	NodeName    string
	State       PodState
	Image       string
	Labels      map[string]string
	Annotations map[string]string
	Ports       []corev1.ContainerPort
	Env         []corev1.EnvVar
	Volumes     []corev1.VolumeMount
	NodeSelector map[string]string
	CreatedAt   int64
}

// Deployment represents a deployment
type Deployment struct {
	Name         string
	Namespace    string
	Replicas     int32
	DesiredReplicas int32
	Pods         []*Pod
	Template     corev1.PodTemplateSpec
	Labels       map[string]string
	Selector     *metav1.LabelSelector
}

// Service represents a Kubernetes service
type Service struct {
	Name        string
	Namespace   string
	Type        corev1.ServiceType
	Selector    map[string]string
	Ports       []corev1.ServicePort
	ClusterIP   string
	Labels      map[string]string
}

// IngressRule represents an ingress rule
type IngressRule struct {
	Host        string
	Paths       []IngressPath
}

// IngressPath represents an ingress path
type IngressPath struct {
	Path        string
	PathType    *networkingv1.PathType
	ServiceName string
	ServicePort int32
}

// Ingress represents a Kubernetes ingress
type Ingress struct {
	Name        string
	Namespace   string
	Rules       []IngressRule
	Labels      map[string]string
	Annotations map[string]string
}

// Node represents a node in the cluster
type Node struct {
	Name        string
	Address     string
	Status      string
	Labels      map[string]string
	Capacity    corev1.ResourceList
	Allocatable corev1.ResourceList
}

// DNSWhitelist represents a DNS whitelist configuration
type DNSWhitelist struct {
	Enabled bool     `json:"enabled"`
	Hosts   []string `json:"hosts"` // List of allowed external hosts/domains
}
