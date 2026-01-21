package main

import (
	"crypto/tls"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/your-server-support/podman-swarm/internal/api"
	"github.com/your-server-support/podman-swarm/internal/cluster"
	"github.com/your-server-support/podman-swarm/internal/config"
	"github.com/your-server-support/podman-swarm/internal/discovery"
	"github.com/your-server-support/podman-swarm/internal/dns"
	"github.com/your-server-support/podman-swarm/internal/ingress"
	"github.com/your-server-support/podman-swarm/internal/parser"
	"github.com/your-server-support/podman-swarm/internal/podman"
	"github.com/your-server-support/podman-swarm/internal/scheduler"
	"github.com/your-server-support/podman-swarm/internal/security"
	"github.com/your-server-support/podman-swarm/internal/storage"
)

func main() {
	cfg := config.Load()

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.Infof("Starting Podman Swarm agent: %s", cfg.NodeName)

	// Initialize token manager
	tokenManager := security.NewTokenManager([]byte(cfg.EncryptionKey))

	// Generate join token if this is the first node and no token provided
	if len(cfg.JoinAddrs) == 0 && cfg.JoinToken == "" {
		token, err := tokenManager.GenerateToken()
		if err != nil {
			logger.Fatalf("Failed to generate join token: %v", err)
		}
		logger.Infof("Generated join token: %s", token)
		logger.Infof("Use this token to join other nodes: --join-token=%s", token)
	}

	// Load TLS configuration if provided
	var tlsConfig *security.TLSConfig
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		tlsConfig = &security.TLSConfig{
			CertFile:   cfg.TLSCertFile,
			KeyFile:    cfg.TLSKeyFile,
			CAFile:     cfg.TLSCAFile,
			SkipVerify: cfg.TLSSkipVerify,
		}
	}

	var tlsConfigLoaded *tls.Config
	if tlsConfig != nil {
		var err error
		tlsConfigLoaded, err = security.LoadTLSConfig(tlsConfig, logger)
		if err != nil {
			logger.Fatalf("Failed to load TLS configuration: %v", err)
		}
		logger.Info("TLS encryption enabled")
	}

	// Prepare encryption key
	var encryptionKey []byte
	if cfg.EncryptionKey != "" {
		encryptionKey = []byte(cfg.EncryptionKey)
	} else if len(cfg.JoinAddrs) > 0 && cfg.JoinToken != "" {
		// If joining, we need the encryption key from the cluster
		// For now, derive from join token
		encryptionKey = []byte(cfg.JoinToken)
	} else {
		// Generate a random key for the first node
		encryptionKey = make([]byte, 32)
		if _, err := os.ReadFile(cfg.DataDir + "/encryption.key"); err == nil {
			// Load existing key if available
			keyData, _ := os.ReadFile(cfg.DataDir + "/encryption.key")
			if len(keyData) >= 32 {
				encryptionKey = keyData[:32]
			}
		} else {
			// Save generated key
			os.MkdirAll(cfg.DataDir, 0755)
			os.WriteFile(cfg.DataDir+"/encryption.key", encryptionKey, 0600)
		}
	}

	// Initialize cluster
	clusterConfig := &cluster.ClusterConfig{
		NodeName:      cfg.NodeName,
		BindAddr:      cfg.BindAddr,
		JoinAddrs:     cfg.JoinAddrs,
		JoinToken:     cfg.JoinToken,
		EncryptionKey: encryptionKey,
		TLSConfig:     tlsConfigLoaded,
		TokenManager:  tokenManager,
		Logger:        logger,
	}

	clusterInstance, err := cluster.NewCluster(clusterConfig)
	if err != nil {
		logger.Fatalf("Failed to initialize cluster: %v", err)
	}
	defer clusterInstance.Shutdown()

	logger.Infof("Cluster initialized with %d nodes", clusterInstance.GetNodeCount())

	// Initialize Podman client
	podmanClient, err := podman.NewClient(cfg.PodmanSocket, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize Podman client: %v", err)
	}

	// Initialize storage
	storageInstance, err := storage.NewStorage(storage.StorageConfig{
		DataDir: cfg.DataDir,
		Logger:  logger,
	})
	if err != nil {
		logger.Fatalf("Failed to initialize storage: %v", err)
	}
	logger.Info("Storage initialized successfully")

	// Initialize service discovery
	discoveryClient := discovery.NewDiscovery(clusterInstance, logger)

	// Set message handler for cluster (handles both service discovery and state sync)
	clusterInstance.SetMessageHandler(func(msg []byte) error {
		// Try to handle as service discovery message first
		discoveryClient.HandleServiceUpdate(msg)
		
		// Try to handle as state sync message
		storageInstance.HandleStateSyncMessage(msg)
		
		// Always return nil - we handle both message types
		return nil
	})

	// Initialize DNS server
	localNodeIP := clusterInstance.GetLocalNodeAddress()
	dnsServer := dns.NewServer(discoveryClient, cfg.ClusterDomain, cfg.DNSPort, localNodeIP, cfg.UpstreamDNS, logger)
	go func() {
		if err := dnsServer.Start(); err != nil {
			logger.Errorf("Failed to start DNS server: %v", err)
		}
	}()
	logger.Infof("DNS server configured. Containers should use DNS: %s", dnsServer.GetDNSIP())

	// Configure Podman client to use DNS server
	podmanClient.SetDNS(dnsServer.GetDNSIP())

	// Initialize API token manager
	apiTokenManager := security.NewAPITokenManager(encryptionKey)
	apiTokenManager.StartCleanupRoutine()

	// Generate initial API token if provided in config
	if cfg.APIToken != "" {
		// Use provided token
		apiTokenManager.GenerateToken("default", nil)
		logger.Infof("Using configured API token")
	} else if cfg.EnableAPIAuth {
		// Generate a new token
		token, err := apiTokenManager.GenerateToken("default", nil)
		if err != nil {
			logger.Fatalf("Failed to generate API token: %v", err)
		}
		logger.Infof("Generated API token: %s", token)
		logger.Infof("Use this token in Authorization header: Bearer %s", token)
	}

	// Initialize scheduler
	schedulerInstance := scheduler.NewScheduler(clusterInstance, logger)

	// Initialize parser
	parserInstance := parser.NewParser()

	// Start periodic backup (every 1 hour)
	storageInstance.StartPeriodicBackup(1 * time.Hour)

	// Start periodic state sync (every 30 seconds)
	storageInstance.StartPeriodicSync(30*time.Second, clusterInstance.Broadcast, clusterInstance.GetLocalNodeName())

	// Initialize ingress controller
	var ingressController *ingress.IngressController
	if cfg.EnableIngress {
		ingressController = ingress.NewIngressController(discoveryClient, cfg.IngressPort, clusterInstance.GetLocalNodeName(), logger)
		go func() {
			if err := ingressController.Start(); err != nil {
				logger.Errorf("Failed to start ingress controller: %v", err)
			}
		}()
	}

	// Initialize API
	apiInstance := api.NewAPI(
		parserInstance,
		schedulerInstance,
		podmanClient,
		discoveryClient,
		ingressController,
		clusterInstance,
		dnsServer,
		storageInstance,
		apiTokenManager,
		logger,
	)

	// Setup API router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// CORS
	corsConfig := cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}
	router.Use(cors.New(corsConfig))

	// Setup routes
	apiInstance.SetupRoutes(router, cfg.EnableAPIAuth)
	
	if cfg.EnableAPIAuth {
		logger.Info("API authentication enabled")
	} else {
		logger.Warn("API authentication disabled - this is not recommended for production")
	}

	// Start state recovery (after cluster is stable)
	apiInstance.StartStateRecovery()

	// Start API server
	go func() {
		logger.Infof("Starting API server on %s", cfg.APIAddr)
		if err := router.Run(cfg.APIAddr); err != nil {
			logger.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
}
