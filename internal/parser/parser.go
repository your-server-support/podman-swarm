package parser

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/your-server-support/podman-swarm/internal/types"
)

type Parser struct {
	decoder runtime.Decoder
}

func NewParser() *Parser {
	sch := scheme.Scheme
	codecs := serializer.NewCodecFactory(sch)

	return &Parser{
		decoder: codecs.UniversalDeserializer(),
	}
}

// ParseManifest parses a Kubernetes manifest YAML
func (p *Parser) ParseManifest(data []byte) ([]runtime.Object, error) {
	// Split by --- for multiple documents
	documents := strings.Split(string(data), "---")
	var objects []runtime.Object

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		obj, _, err := p.decoder.Decode([]byte(doc), nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decode manifest: %w", err)
		}

		objects = append(objects, obj)
	}

	return objects, nil
}

// ParseDeployment extracts deployment information
func (p *Parser) ParseDeployment(obj runtime.Object) (*types.Deployment, error) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil, fmt.Errorf("object is not a Deployment")
	}

	dep := &types.Deployment{
		Name:            deployment.Name,
		Namespace:       deployment.Namespace,
		Replicas:        *deployment.Spec.Replicas,
		DesiredReplicas: *deployment.Spec.Replicas,
		Template:        deployment.Spec.Template,
		Labels:          deployment.Labels,
		Selector:        deployment.Spec.Selector,
		Pods:            []*types.Pod{},
	}

	return dep, nil
}

// ParseService extracts service information
func (p *Parser) ParseService(obj runtime.Object) (*types.Service, error) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("object is not a Service")
	}

	svc := &types.Service{
		Name:      service.Name,
		Namespace: service.Namespace,
		Type:      service.Spec.Type,
		Selector:  service.Spec.Selector,
		Ports:     service.Spec.Ports,
		ClusterIP: service.Spec.ClusterIP,
		Labels:    service.Labels,
	}

	return svc, nil
}

// ParseIngress extracts ingress information
func (p *Parser) ParseIngress(obj runtime.Object) (*types.Ingress, error) {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return nil, fmt.Errorf("object is not an Ingress")
	}

	ing := &types.Ingress{
		Name:        ingress.Name,
		Namespace:   ingress.Namespace,
		Labels:      ingress.Labels,
		Annotations: ingress.Annotations,
		Rules:       []types.IngressRule{},
	}

	for _, rule := range ingress.Spec.Rules {
		ingRule := types.IngressRule{
			Host:  rule.Host,
			Paths: []types.IngressPath{},
		}

		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				ingPath := types.IngressPath{
					Path:     path.Path,
					PathType: path.PathType,
				}

				if path.Backend.Service != nil {
					ingPath.ServiceName = path.Backend.Service.Name
					if path.Backend.Service.Port.Number != 0 {
						ingPath.ServicePort = path.Backend.Service.Port.Number
					} else {
						// If Port.Name is specified, we'll use 0 and resolve it later
						// For now, default to port 80
						ingPath.ServicePort = 80
					}
				}

				ingRule.Paths = append(ingRule.Paths, ingPath)
			}
		}

		ing.Rules = append(ing.Rules, ingRule)
	}

	return ing, nil
}

// ExtractPodFromTemplate creates a Pod from a PodTemplateSpec
func (p *Parser) ExtractPodFromTemplate(template corev1.PodTemplateSpec, namespace, podName string) *types.Pod {
	pod := &types.Pod{
		Name:        podName,
		Namespace:   namespace,
		Labels:      template.Labels,
		Annotations: template.Annotations,
		State:       types.PodStatePending,
	}

	if len(template.Spec.Containers) > 0 {
		container := template.Spec.Containers[0]
		pod.Image = container.Image
		pod.Ports = container.Ports
		pod.Env = container.Env

		// Convert volume mounts
		for _, vm := range container.VolumeMounts {
			pod.Volumes = append(pod.Volumes, corev1.VolumeMount{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				ReadOnly:  vm.ReadOnly,
			})
		}
	}

	// Extract node selector
	pod.NodeSelector = template.Spec.NodeSelector

	return pod
}

// ParseYAML parses YAML directly (fallback)
func (p *Parser) ParseYAML(data []byte, obj interface{}) error {
	return yaml.Unmarshal(data, obj)
}
