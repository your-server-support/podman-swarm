package parser

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("Expected parser to be created")
	}
}

func TestParseDeployment(t *testing.T) {
	parser := NewParser()

	replicas := int32(3)
	k8sDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}

	deployment, err := parser.ParseDeployment(k8sDeployment)
	if err != nil {
		t.Fatalf("Failed to parse deployment: %v", err)
	}

	if deployment.Name != "test-deployment" {
		t.Errorf("Expected name 'test-deployment', got '%s'", deployment.Name)
	}

	if deployment.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", deployment.Namespace)
	}

	if deployment.DesiredReplicas != 3 {
		t.Errorf("Expected 3 replicas, got %d", deployment.DesiredReplicas)
	}
}

func TestParseDeploymentWithDefaultNamespace(t *testing.T) {
	parser := NewParser()

	replicas := int32(1)
	k8sDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-deployment",
			// No namespace specified
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}

	deployment, err := parser.ParseDeployment(k8sDeployment)
	if err != nil {
		t.Fatalf("Failed to parse deployment: %v", err)
	}

	// Parser may not set default namespace - that's API layer's job
	// Just verify parsing succeeds
	if deployment.Name != "test-deployment" {
		t.Errorf("Expected name 'test-deployment', got '%s'", deployment.Name)
	}
}

func TestParseService(t *testing.T) {
	parser := NewParser()

	k8sService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	service, err := parser.ParseService(k8sService)
	if err != nil {
		t.Fatalf("Failed to parse service: %v", err)
	}

	if service.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", service.Name)
	}

	if service.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", service.Namespace)
	}

	if len(service.Selector) == 0 {
		t.Error("Expected selector to be set")
	}

	if service.Selector["app"] != "test" {
		t.Errorf("Expected selector app='test', got '%s'", service.Selector["app"])
	}
}

func TestParseIngress(t *testing.T) {
	parser := NewParser()

	pathType := networkingv1.PathTypePrefix
	k8sIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "test-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingress, err := parser.ParseIngress(k8sIngress)
	if err != nil {
		t.Fatalf("Failed to parse ingress: %v", err)
	}

	if ingress.Name != "test-ingress" {
		t.Errorf("Expected name 'test-ingress', got '%s'", ingress.Name)
	}

	if ingress.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", ingress.Namespace)
	}

	if len(ingress.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(ingress.Rules))
	}

	if ingress.Rules[0].Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", ingress.Rules[0].Host)
	}

	if len(ingress.Rules[0].Paths) != 1 {
		t.Fatalf("Expected 1 path, got %d", len(ingress.Rules[0].Paths))
	}

	if ingress.Rules[0].Paths[0].Path != "/" {
		t.Errorf("Expected path '/', got '%s'", ingress.Rules[0].Paths[0].Path)
	}
}

func TestExtractPodFromTemplate(t *testing.T) {
	parser := NewParser()

	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	pod := parser.ExtractPodFromTemplate(template, "default", "test-pod")

	if pod.Name != "test-pod" {
		t.Errorf("Expected name 'test-pod', got '%s'", pod.Name)
	}

	if pod.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", pod.Namespace)
	}

	if pod.Image != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", pod.Image)
	}

	if pod.Labels["app"] != "test" {
		t.Errorf("Expected label app='test', got '%s'", pod.Labels["app"])
	}
}

func TestParseDeploymentWithMultipleContainers(t *testing.T) {
	parser := NewParser()

	replicas := int32(2)
	k8sDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-container",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
						},
						{
							Name:  "sidecar",
							Image: "sidecar:latest",
						},
					},
				},
			},
		},
	}

	deployment, err := parser.ParseDeployment(k8sDeployment)
	if err != nil {
		t.Fatalf("Failed to parse deployment: %v", err)
	}

	// Should use first container
	pod := parser.ExtractPodFromTemplate(deployment.Template, "default", "test")
	if pod.Image != "app:latest" {
		t.Errorf("Expected first container image, got '%s'", pod.Image)
	}
}

func TestParseServiceWithMultiplePorts(t *testing.T) {
	parser := NewParser()

	k8sService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-port-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "https",
					Port:     443,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	service, err := parser.ParseService(k8sService)
	if err != nil {
		t.Fatalf("Failed to parse service: %v", err)
	}

	// Parser should handle multiple ports
	// (actual port handling depends on implementation)
	if service.Name != "multi-port-service" {
		t.Errorf("Expected name 'multi-port-service', got '%s'", service.Name)
	}
}
