package cluster

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/hashicorp/memberlist"
	"github.com/sirupsen/logrus"

	"github.com/your-server-support/podman-swarm/internal/security"
	"github.com/your-server-support/podman-swarm/internal/types"
)

type MessageHandler func([]byte) error

type Cluster struct {
	memberlist     *memberlist.Memberlist
	delegate       *delegate
	mu             sync.RWMutex
	nodes          map[string]*types.Node
	logger         *logrus.Logger
	messageHandler MessageHandler
	encryptor      *security.Encryptor
	tokenManager   *security.TokenManager
	tlsConfig      *tls.Config
}

type delegate struct {
	cluster *Cluster
	logger  *logrus.Logger
}

func (d *delegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *delegate) NotifyMsg(msg []byte) {
	// Decrypt message if encryption is enabled
	decryptedMsg := msg
	if d.cluster.encryptor != nil {
		decrypted, err := d.cluster.encryptor.Decrypt(msg)
		if err != nil {
			d.logger.Warnf("Failed to decrypt message: %v", err)
			return
		}
		decryptedMsg = decrypted
	}

	d.logger.Debugf("Received message: %s", string(decryptedMsg))
	if d.cluster.messageHandler != nil {
		if err := d.cluster.messageHandler(decryptedMsg); err != nil {
			d.logger.Warnf("Error handling message: %v", err)
		}
	}
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (d *delegate) LocalState(join bool) []byte {
	return []byte{}
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {
}

func (d *delegate) NotifyJoin(node *memberlist.Node) {
	d.cluster.mu.Lock()
	defer d.cluster.mu.Unlock()

	// Validate token if token manager is set
	// Token validation happens during join process, not here
	// This is just for logging

	d.logger.Infof("Node %s joined the cluster", node.Name)
	d.cluster.nodes[node.Name] = &types.Node{
		Name:    node.Name,
		Address: node.Addr.String(),
		Status:  "Ready",
		Labels:  make(map[string]string),
	}
}

func (d *delegate) NotifyLeave(node *memberlist.Node) {
	d.cluster.mu.Lock()
	defer d.cluster.mu.Unlock()

	d.logger.Infof("Node %s left the cluster", node.Name)
	delete(d.cluster.nodes, node.Name)
}

func (d *delegate) NotifyUpdate(node *memberlist.Node) {
	d.cluster.mu.Lock()
	defer d.cluster.mu.Unlock()

	if existing, ok := d.cluster.nodes[node.Name]; ok {
		existing.Address = node.Addr.String()
	}
}

type ClusterConfig struct {
	NodeName      string
	BindAddr      string
	JoinAddrs     []string
	JoinToken     string
	EncryptionKey []byte
	TLSConfig     *tls.Config
	TokenManager  *security.TokenManager
	Logger        *logrus.Logger
}

func NewCluster(cfg *ClusterConfig) (*Cluster, error) {
	config := memberlist.DefaultLocalConfig()
	config.Name = cfg.NodeName
	config.BindAddr = cfg.BindAddr
	config.BindPort = 7946
	config.AdvertiseAddr = cfg.BindAddr
	config.AdvertisePort = 7946
	config.LogOutput = cfg.Logger.Writer()

	cluster := &Cluster{
		nodes:        make(map[string]*types.Node),
		logger:       cfg.Logger,
		tokenManager: cfg.TokenManager,
		tlsConfig:    cfg.TLSConfig,
	}

	// Setup encryption if key is provided
	if len(cfg.EncryptionKey) > 0 {
		encryptor, err := security.NewEncryptor(cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
		cluster.encryptor = encryptor
	}

	// Setup TLS transport if TLS config is provided
	// Note: memberlist doesn't directly support custom transport in v0.5.0
	// For now, we'll use encryption at the message level
	// TLS can be added via network-level encryption (VPN, etc.) or by wrapping connections
	// For production, consider using memberlist with custom transport implementation

	cluster.delegate = &delegate{
		cluster: cluster,
		logger:  cfg.Logger,
	}

	config.Delegate = cluster.delegate
	config.Events = cluster.delegate

	list, err := memberlist.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create memberlist: %w", err)
	}

	cluster.memberlist = list

	// Validate join token if provided
	if len(cfg.JoinAddrs) > 0 {
		if cfg.TokenManager != nil && cfg.JoinToken != "" {
			if !cfg.TokenManager.ValidateToken(cfg.JoinToken) {
				return nil, fmt.Errorf("invalid join token")
			}
			cfg.Logger.Infof("Join token validated successfully")
		}

		_, err := list.Join(cfg.JoinAddrs)
		if err != nil {
			cfg.Logger.Warnf("Failed to join cluster: %v", err)
		} else {
			cfg.Logger.Infof("Joined cluster with %d nodes", len(list.Members()))
		}
	}

	// Add local node
	cluster.mu.Lock()
	cluster.nodes[cfg.NodeName] = &types.Node{
		Name:    cfg.NodeName,
		Address: cfg.BindAddr,
		Status:  "Ready",
		Labels:  make(map[string]string),
	}
	cluster.mu.Unlock()

	return cluster, nil
}

func (c *Cluster) GetNodes() []*types.Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]*types.Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

func (c *Cluster) GetNode(name string) (*types.Node, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, ok := c.nodes[name]
	if !ok {
		return nil, fmt.Errorf("node %s not found", name)
	}

	return node, nil
}

func (c *Cluster) GetMembers() []*memberlist.Node {
	return c.memberlist.Members()
}

func (c *Cluster) Broadcast(msg []byte) error {
	// Encrypt message if encryption is enabled
	sendMsg := msg
	if c.encryptor != nil {
		encrypted, err := c.encryptor.Encrypt(msg)
		if err != nil {
			return fmt.Errorf("failed to encrypt message: %w", err)
		}
		sendMsg = encrypted
	}

	return c.memberlist.SendBestEffort(nil, sendMsg)
}

func (c *Cluster) Shutdown() error {
	return c.memberlist.Shutdown()
}

func (c *Cluster) UpdateNodeLabels(nodeName string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.nodes[nodeName]; ok {
		node.Labels = labels
	}
}

func (c *Cluster) GetNodeCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.nodes)
}

func (c *Cluster) GetLocalNodeName() string {
	return c.memberlist.LocalNode().Name
}

// GetLocalNodeAddress returns the IP address of the local node
func (c *Cluster) GetLocalNodeAddress() string {
	localNode := c.memberlist.LocalNode()
	if localNode != nil {
		return localNode.Addr.String()
	}
	// Fallback: try to get from nodes map
	c.mu.RLock()
	defer c.mu.RUnlock()
	if localNode, err := c.GetNode(c.memberlist.LocalNode().Name); err == nil {
		return localNode.Address
	}
	return "127.0.0.1"
}

func (c *Cluster) SetMessageHandler(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageHandler = handler
}

func (c *Cluster) IsLeader() bool {
	// In a peer-to-peer cluster, any node can be a leader
	// For simplicity, we can use the first node alphabetically
	members := c.memberlist.Members()
	if len(members) == 0 {
		return false
	}

	localName := c.memberlist.LocalNode().Name
	for _, member := range members {
		if member.Name < localName {
			return false
		}
	}
	return true
}
