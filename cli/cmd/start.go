package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/aqz236/port-fly/core/manager"
	"github.com/aqz236/port-fly/core/models"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [flags] [user@]hostname",
	Short: "Start a new SSH tunnel session",
	Long: `Start a new SSH tunnel session with the specified configuration.

Examples:
  # Local port forwarding (SSH -L)
  portfly start -L 8080:192.168.1.100:80 user@example.com
  portfly start -L 127.0.0.1:8080:192.168.1.100:80 user@example.com
  
  # Remote port forwarding (SSH -R)
  portfly start -R 8080:localhost:3000 user@example.com
  portfly start -R 0.0.0.0:8080:localhost:3000 user@example.com
  
  # Dynamic port forwarding / SOCKS proxy (SSH -D)
  portfly start -D 1080 user@example.com
  portfly start -D 127.0.0.1:1080 user@example.com
  
  # Multiple tunnels in one session
  portfly start -L 8080:web:80 -L 3306:db:3306 -D 1080 user@example.com
  
  # With authentication options
  portfly start -L 8080:web:80 -i ~/.ssh/id_rsa user@example.com
  portfly start -L 8080:web:80 --password user@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

var (
	// Tunnel flags
	localForwards   []string
	remoteForwards  []string
	dynamicForwards []string

	// SSH connection flags
	sshPort      int
	identityFile string
	password     bool
	authMethod   string

	// Tunnel options
	sessionName    string
	sessionDesc    string
	background     bool
	keepAlive      time.Duration
	connectTimeout time.Duration
	maxRetries     int
	retryInterval  time.Duration
)

func init() {
	rootCmd.AddCommand(startCmd)

	// Tunnel configuration flags
	startCmd.Flags().StringSliceVarP(&localForwards, "local-forward", "L", []string{},
		"Local port forwarding: [bind_address:]port:host:hostport")
	startCmd.Flags().StringSliceVarP(&remoteForwards, "remote-forward", "R", []string{},
		"Remote port forwarding: [bind_address:]port:host:hostport")
	startCmd.Flags().StringSliceVarP(&dynamicForwards, "dynamic", "D", []string{},
		"Dynamic port forwarding (SOCKS): [bind_address:]port")

	// SSH connection flags
	startCmd.Flags().IntVarP(&sshPort, "port", "p", 22, "SSH port")
	startCmd.Flags().StringVarP(&identityFile, "identity", "i", "", "Path to private key file")
	startCmd.Flags().BoolVar(&password, "password", false, "Use password authentication (will prompt)")
	startCmd.Flags().StringVar(&authMethod, "auth-method", "", "Authentication method: password, private_key, agent")

	// Session options
	startCmd.Flags().StringVarP(&sessionName, "name", "n", "", "Session name (auto-generated if not specified)")
	startCmd.Flags().StringVarP(&sessionDesc, "description", "d", "", "Session description")
	startCmd.Flags().BoolVarP(&background, "background", "b", false, "Run in background (daemon mode)")

	// Connection options
	startCmd.Flags().DurationVar(&keepAlive, "keep-alive", 30*time.Second, "SSH keep-alive interval")
	startCmd.Flags().DurationVar(&connectTimeout, "connect-timeout", 30*time.Second, "SSH connection timeout")
	startCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "Maximum connection retry attempts")
	startCmd.Flags().DurationVar(&retryInterval, "retry-interval", 5*time.Second, "Retry interval")
}

func runStart(cmd *cobra.Command, args []string) error {
	logger.Info("starting new SSH tunnel session")

	// Parse target host
	target := args[0]
	sshConfig, err := parseSSHTarget(target)
	if err != nil {
		return fmt.Errorf("invalid SSH target: %w", err)
	}

	// Override with command line flags
	if sshPort != 22 {
		sshConfig.Port = sshPort
	}
	if identityFile != "" {
		sshConfig.PrivateKeyPath = identityFile
		sshConfig.AuthMethod = models.AuthMethodPrivateKey
	}
	if password {
		sshConfig.AuthMethod = models.AuthMethodPassword
		// Prompt for password securely
		fmt.Print("Enter SSH password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println() // New line after password input
		sshConfig.Password = string(passwordBytes)
	}
	if authMethod != "" {
		switch strings.ToLower(authMethod) {
		case "password":
			sshConfig.AuthMethod = models.AuthMethodPassword
			// Prompt for password if not already provided
			if sshConfig.Password == "" {
				fmt.Print("Enter SSH password: ")
				passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}
				fmt.Println() // New line after password input
				sshConfig.Password = string(passwordBytes)
			}
		case "private_key", "key":
			sshConfig.AuthMethod = models.AuthMethodPrivateKey
		case "agent":
			sshConfig.AuthMethod = models.AuthMethodAgent
		default:
			return fmt.Errorf("invalid auth method: %s", authMethod)
		}
	}

	// Set connection options
	sshConfig.ConnectTimeout = connectTimeout
	sshConfig.KeepAliveTimeout = keepAlive
	sshConfig.MaxRetries = maxRetries
	sshConfig.RetryInterval = retryInterval

	// Parse tunnel configurations
	tunnelConfigs, err := parseTunnelConfigs()
	if err != nil {
		return fmt.Errorf("invalid tunnel configuration: %w", err)
	}

	if len(tunnelConfigs) == 0 {
		return fmt.Errorf("no tunnel configurations specified")
	}

	// Create session manager
	sessionMgr := manager.NewSessionManager(config.SSH, logger)

	// Create sessions for each tunnel
	var sessionIDs []string
	for i, tunnelConfig := range tunnelConfigs {
		session, err := sessionMgr.CreateSession(sshConfig, tunnelConfig)
		if err != nil {
			return fmt.Errorf("failed to create session %d: %w", i+1, err)
		}

		// Set session name and description
		if sessionName != "" {
			if len(tunnelConfigs) > 1 {
				session.Name = fmt.Sprintf("%s-%d", sessionName, i+1)
			} else {
				session.Name = sessionName
			}
		}
		if sessionDesc != "" {
			session.Description = sessionDesc
		}

		sessionIDs = append(sessionIDs, session.ID)

		// Start the session
		if err := sessionMgr.StartSession(session.ID); err != nil {
			return fmt.Errorf("failed to start session %s: %w", session.ID, err)
		}

		logger.Info("session started",
			"session_id", session.ID,
			"name", session.Name,
			"description", session.Description)
	}

	if !background {
		// Wait for sessions in foreground mode
		fmt.Printf("Started %d tunnel session(s). Press Ctrl+C to stop.\n", len(sessionIDs))
		for _, sessionID := range sessionIDs {
			fmt.Printf("  Session: %s\n", sessionID)
		}

		// TODO: Add signal handling to gracefully stop sessions
		// For now, just wait
		select {}
	} else {
		// Background mode - just print session IDs
		fmt.Printf("Started %d tunnel session(s) in background:\n", len(sessionIDs))
		for _, sessionID := range sessionIDs {
			fmt.Printf("  Session: %s\n", sessionID)
		}
	}

	return nil
}

// parseSSHTarget parses user@hostname format
func parseSSHTarget(target string) (models.SSHConnectionConfig, error) {
	config := models.SSHConnectionConfig{
		Port:            22,
		AuthMethod:      models.AuthMethodPrivateKey, // Default
		HostKeyCallback: "ask",
	}

	parts := strings.Split(target, "@")
	if len(parts) == 2 {
		config.Username = parts[0]
		config.Host = parts[1]
	} else {
		config.Host = target
		// Use current user as default
		if currentUser := os.Getenv("USER"); currentUser != "" {
			config.Username = currentUser
		} else {
			return config, fmt.Errorf("username not specified and USER environment variable not set")
		}
	}

	// Parse host:port if specified
	if strings.Contains(config.Host, ":") {
		hostParts := strings.Split(config.Host, ":")
		if len(hostParts) == 2 {
			config.Host = hostParts[0]
			port, err := strconv.Atoi(hostParts[1])
			if err != nil {
				return config, fmt.Errorf("invalid port: %s", hostParts[1])
			}
			config.Port = port
		}
	}

	return config, nil
}

// parseTunnelConfigs parses tunnel configuration flags
func parseTunnelConfigs() ([]models.TunnelConfig, error) {
	var configs []models.TunnelConfig

	// Parse local forwards (-L)
	for _, forward := range localForwards {
		config, err := parseLocalForward(forward)
		if err != nil {
			return nil, fmt.Errorf("invalid local forward '%s': %w", forward, err)
		}
		configs = append(configs, config)
	}

	// Parse remote forwards (-R)
	for _, forward := range remoteForwards {
		config, err := parseRemoteForward(forward)
		if err != nil {
			return nil, fmt.Errorf("invalid remote forward '%s': %w", forward, err)
		}
		configs = append(configs, config)
	}

	// Parse dynamic forwards (-D)
	for _, forward := range dynamicForwards {
		config, err := parseDynamicForward(forward)
		if err != nil {
			return nil, fmt.Errorf("invalid dynamic forward '%s': %w", forward, err)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// parseLocalForward parses local forward specification
func parseLocalForward(spec string) (models.TunnelConfig, error) {
	config := models.TunnelConfig{
		Type:             models.TunnelTypeLocal,
		LocalBindAddress: "127.0.0.1",
	}

	// Format: [bind_address:]port:host:hostport
	parts := strings.Split(spec, ":")

	switch len(parts) {
	case 3:
		// port:host:hostport
		localPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return config, fmt.Errorf("invalid local port: %s", parts[0])
		}
		remotePort, err := strconv.Atoi(parts[2])
		if err != nil {
			return config, fmt.Errorf("invalid remote port: %s", parts[2])
		}

		config.LocalPort = localPort
		config.RemoteHost = parts[1]
		config.RemotePort = remotePort

	case 4:
		// bind_address:port:host:hostport
		localPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return config, fmt.Errorf("invalid local port: %s", parts[1])
		}
		remotePort, err := strconv.Atoi(parts[3])
		if err != nil {
			return config, fmt.Errorf("invalid remote port: %s", parts[3])
		}

		config.LocalBindAddress = parts[0]
		config.LocalPort = localPort
		config.RemoteHost = parts[2]
		config.RemotePort = remotePort

	default:
		return config, fmt.Errorf("invalid format, expected [bind_address:]port:host:hostport")
	}

	return config, nil
}

// parseRemoteForward parses remote forward specification
func parseRemoteForward(spec string) (models.TunnelConfig, error) {
	config := models.TunnelConfig{
		Type:              models.TunnelTypeRemote,
		RemoteBindAddress: "127.0.0.1",
	}

	// Format: [bind_address:]port:host:hostport
	parts := strings.Split(spec, ":")

	switch len(parts) {
	case 3:
		// port:host:hostport
		remotePort, err := strconv.Atoi(parts[0])
		if err != nil {
			return config, fmt.Errorf("invalid remote port: %s", parts[0])
		}
		localPort, err := strconv.Atoi(parts[2])
		if err != nil {
			return config, fmt.Errorf("invalid local port: %s", parts[2])
		}

		config.LocalPort = remotePort // Remote port becomes local port in our model
		config.RemoteHost = parts[1]
		config.RemotePort = localPort

	case 4:
		// bind_address:port:host:hostport
		remotePort, err := strconv.Atoi(parts[1])
		if err != nil {
			return config, fmt.Errorf("invalid remote port: %s", parts[1])
		}
		localPort, err := strconv.Atoi(parts[3])
		if err != nil {
			return config, fmt.Errorf("invalid local port: %s", parts[3])
		}

		config.RemoteBindAddress = parts[0]
		config.LocalPort = remotePort
		config.RemoteHost = parts[2]
		config.RemotePort = localPort

	default:
		return config, fmt.Errorf("invalid format, expected [bind_address:]port:host:hostport")
	}

	return config, nil
}

// parseDynamicForward parses dynamic forward specification
func parseDynamicForward(spec string) (models.TunnelConfig, error) {
	config := models.TunnelConfig{
		Type:             models.TunnelTypeDynamic,
		SOCKSBindAddress: "127.0.0.1",
		SOCKSVersion:     5, // Default to SOCKS5
	}

	// Format: [bind_address:]port
	parts := strings.Split(spec, ":")

	switch len(parts) {
	case 1:
		// port only
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			return config, fmt.Errorf("invalid port: %s", parts[0])
		}
		config.SOCKSPort = port

	case 2:
		// bind_address:port
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return config, fmt.Errorf("invalid port: %s", parts[1])
		}
		config.SOCKSBindAddress = parts[0]
		config.SOCKSPort = port

	default:
		return config, fmt.Errorf("invalid format, expected [bind_address:]port")
	}

	return config, nil
}
