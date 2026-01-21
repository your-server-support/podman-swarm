package podman

import (
	"context"
	"fmt"
	"io"

	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"

	"github.com/your-server-support/podman-swarm/internal/types"
)

type Client struct {
	conn   context.Context
	logger *logrus.Logger
	dnsIP  string // DNS server IP address for containers
}

func NewClient(socket string, logger *logrus.Logger) (*Client, error) {
	ctx, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to podman: %w", err)
	}

	return &Client{
		conn:   ctx,
		logger: logger,
	}, nil
}

// SetDNS sets the DNS server IP address for containers
func (c *Client) SetDNS(dnsIP string) {
	c.dnsIP = dnsIP
}

func (c *Client) CreatePod(pod *types.Pod) (string, error) {
	// Create specgen spec for container
	s := specgen.NewSpecGenerator(pod.Image, false)
	s.Name = pod.Name

	// Set environment variables
	env := make(map[string]string)
	for _, e := range pod.Env {
		env[e.Name] = e.Value
	}
	s.Env = env

	// Set labels
	s.Labels = pod.Labels

	// Set network namespace to bridge (default)
	netNS, _, _, err := specgen.ParseNetworkFlag([]string{"bridge"}, false)
	if err == nil {
		s.NetNS = netNS
	} else {
		// Fallback to default bridge network
		s.NetNS = specgen.Namespace{
			NSMode: specgen.Bridge,
		}
	}

	// Set port mappings
	portMappings := []nettypes.PortMapping{}
	for _, port := range pod.Ports {
		hostPort := port.HostPort
		if hostPort == 0 {
			hostPort = port.ContainerPort
		}

		protocol := "tcp"
		if port.Protocol != "" {
			protocol = string(port.Protocol)
		}

		portMapping := nettypes.PortMapping{
			ContainerPort: uint16(port.ContainerPort),
			HostPort:      uint16(hostPort),
			Protocol:      protocol,
		}

		if port.HostIP != "" {
			portMapping.HostIP = port.HostIP
		}

		portMappings = append(portMappings, portMapping)
	}
	s.PortMappings = portMappings

	// Set volume mounts
	if len(pod.Volumes) > 0 {
		mounts := []specs.Mount{}
		for _, vol := range pod.Volumes {
			options := []string{}
			if vol.ReadOnly {
				options = append(options, "ro")
			}

			mount := specs.Mount{
				Type:        "bind",
				Destination: vol.MountPath,
				Options:     options,
			}

			// Note: Source should be set from pod spec
			// For now, we'll use the mount path as source if not specified
			// In a full implementation, you'd map volumes from pod spec
			if vol.Name != "" {
				// If volume name is provided, use it as source
				mount.Source = vol.Name
			} else {
				mount.Source = vol.MountPath
			}

			mounts = append(mounts, mount)
		}
		s.Mounts = mounts
	}

	// Set DNS servers if configured
	if c.dnsIP != "" {
		s.DNSServers = []string{c.dnsIP}
		c.logger.Debugf("Setting DNS server for container %s: %s", pod.Name, c.dnsIP)
	}

	// Create container using specgen
	response, err := containers.CreateWithSpec(c.conn, s, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	c.logger.Infof("Created container %s (ID: %s)", pod.Name, response.ID)

	return response.ID, nil
}

func (c *Client) StartPod(containerID string) error {
	return containers.Start(c.conn, containerID, nil)
}

func (c *Client) StopPod(containerID string) error {
	timeout := uint(10)
	return containers.Stop(c.conn, containerID, &containers.StopOptions{
		Timeout: &timeout,
	})
}

func (c *Client) RemovePod(containerID string) error {
	force := true
	_, err := containers.Remove(c.conn, containerID, &containers.RemoveOptions{
		Force: &force,
	})
	return err
}

func (c *Client) GetPodStatus(containerID string) (types.PodState, error) {
	data, err := containers.Inspect(c.conn, containerID, nil)
	if err != nil {
		return types.PodStateUnknown, err
	}

	if data.State != nil {
		switch data.State.Status {
		case "running":
			return types.PodStateRunning, nil
		case "exited":
			if data.State.ExitCode == 0 {
				return types.PodStateSucceeded, nil
			}
			return types.PodStateFailed, nil
		case "created", "configured":
			return types.PodStatePending, nil
		}
	}

	return types.PodStateUnknown, nil
}

func (c *Client) ListPods() ([]entities.ListContainer, error) {
	return containers.List(c.conn, &containers.ListOptions{
		All: &[]bool{true}[0],
	})
}

func (c *Client) PullImage(image string) error {
	_, err := images.Pull(c.conn, image, &images.PullOptions{})
	return err
}

func (c *Client) GetLogs(containerID string, follow bool) (io.ReadCloser, error) {
	// Podman v4 logs implementation
	// Create channels for stdout and stderr
	stdout := make(chan string, 1)
	stderr := make(chan string, 1)

	// Start logs streaming
	errChan := make(chan error, 1)
	go func() {
		err := containers.Logs(c.conn, containerID, &containers.LogOptions{
			Follow: &follow,
			Stdout: &[]bool{true}[0],
			Stderr: &[]bool{true}[0],
		}, stdout, stderr)
		if err != nil {
			errChan <- err
		}
		close(stdout)
		close(stderr)
	}()

	// Create a reader that combines stdout and stderr
	reader := &combinedLogReader{
		stdout:  stdout,
		stderr:  stderr,
		errChan: errChan,
	}

	return reader, nil
}

func (c *Client) Exec(containerID string, command []string) (string, error) {
	// Podman exec implementation
	// This would use containers.ExecCreate and containers.ExecStart
	return "", fmt.Errorf("exec not yet implemented")
}

// combinedLogReader combines stdout and stderr channels into a single ReadCloser
type combinedLogReader struct {
	stdout  chan string
	stderr  chan string
	errChan chan error
	buffer  []byte
	closed  bool
}

func (r *combinedLogReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, io.EOF
	}

	if len(r.buffer) == 0 {
		// Try to read from channels
		select {
		case line, ok := <-r.stdout:
			if !ok {
				r.stdout = nil
			} else {
				r.buffer = append(r.buffer, []byte(line+"\n")...)
			}
		case line, ok := <-r.stderr:
			if !ok {
				r.stderr = nil
			} else {
				r.buffer = append(r.buffer, []byte(line+"\n")...)
			}
		case err := <-r.errChan:
			if err != nil {
				return 0, err
			}
		}

		if r.stdout == nil && r.stderr == nil {
			r.closed = true
			if len(r.buffer) == 0 {
				return 0, io.EOF
			}
		}
	}

	n = copy(p, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

func (r *combinedLogReader) Close() error {
	r.closed = true
	return nil
}
