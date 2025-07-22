package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/aqz236/port-fly/core/models"
)

// AuthProvider defines the interface for SSH authentication providers
type AuthProvider interface {
	GetAuthMethods(config models.SSHConnectionConfig) ([]ssh.AuthMethod, error)
	GetName() string
}

// PasswordAuthProvider implements password authentication
type PasswordAuthProvider struct{}

// GetAuthMethods returns password authentication method
func (p *PasswordAuthProvider) GetAuthMethods(config models.SSHConnectionConfig) ([]ssh.AuthMethod, error) {
	if config.Password == "" {
		return nil, fmt.Errorf("password is required for password authentication")
	}
	
	return []ssh.AuthMethod{
		ssh.Password(config.Password),
	}, nil
}

// GetName returns the provider name
func (p *PasswordAuthProvider) GetName() string {
	return "password"
}

// PrivateKeyAuthProvider implements private key authentication
type PrivateKeyAuthProvider struct {
	cryptoUtils *CryptoUtils
}

// NewPrivateKeyAuthProvider creates a new private key auth provider
func NewPrivateKeyAuthProvider() *PrivateKeyAuthProvider {
	return &PrivateKeyAuthProvider{
		cryptoUtils: NewCryptoUtils(),
	}
}

// GetAuthMethods returns private key authentication method
func (p *PrivateKeyAuthProvider) GetAuthMethods(config models.SSHConnectionConfig) ([]ssh.AuthMethod, error) {
	var signer ssh.Signer
	var err error
	
	// Try to load from provided key data first
	if len(config.PrivateKeyData) > 0 {
		signer, err = p.cryptoUtils.LoadPrivateKey(config.PrivateKeyData, config.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to load private key from data: %w", err)
		}
	} else if config.PrivateKeyPath != "" {
		// Load from file
		signer, err = p.cryptoUtils.LoadPrivateKeyFromFile(config.PrivateKeyPath, config.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to load private key from file %s: %w", config.PrivateKeyPath, err)
		}
	} else {
		// Try to find default keys
		availableKeys := p.cryptoUtils.FindAvailableKeys()
		if len(availableKeys) == 0 {
			return nil, fmt.Errorf("no private keys found")
		}
		
		// Try each available key
		for _, keyPath := range availableKeys {
			signer, err = p.cryptoUtils.LoadPrivateKeyFromFile(keyPath, config.Passphrase)
			if err == nil {
				break
			}
		}
		
		if signer == nil {
			return nil, fmt.Errorf("failed to load any available private keys")
		}
	}
	
	return []ssh.AuthMethod{
		ssh.PublicKeys(signer),
	}, nil
}

// GetName returns the provider name
func (p *PrivateKeyAuthProvider) GetName() string {
	return "private_key"
}

// AgentAuthProvider implements SSH agent authentication
type AgentAuthProvider struct{}

// GetAuthMethods returns SSH agent authentication method
func (a *AgentAuthProvider) GetAuthMethods(config models.SSHConnectionConfig) ([]ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}
	
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	
	agentClient := agent.NewClient(conn)
	
	return []ssh.AuthMethod{
		ssh.PublicKeysCallback(agentClient.Signers),
	}, nil
}

// GetName returns the provider name
func (a *AgentAuthProvider) GetName() string {
	return "agent"
}

// InteractiveAuthProvider implements keyboard-interactive authentication
type InteractiveAuthProvider struct {
	challengeHandler func(name, instruction string, questions []string, echos []bool) ([]string, error)
}

// NewInteractiveAuthProvider creates a new interactive auth provider
func NewInteractiveAuthProvider(handler func(string, string, []string, []bool) ([]string, error)) *InteractiveAuthProvider {
	return &InteractiveAuthProvider{
		challengeHandler: handler,
	}
}

// GetAuthMethods returns keyboard-interactive authentication method
func (i *InteractiveAuthProvider) GetAuthMethods(config models.SSHConnectionConfig) ([]ssh.AuthMethod, error) {
	if i.challengeHandler == nil {
		return nil, fmt.Errorf("no challenge handler provided for interactive authentication")
	}
	
	return []ssh.AuthMethod{
		ssh.KeyboardInteractive(i.challengeHandler),
	}, nil
}

// GetName returns the provider name
func (i *InteractiveAuthProvider) GetName() string {
	return "interactive"
}

// AuthManager manages SSH authentication
type AuthManager struct {
	providers map[models.AuthMethod]AuthProvider
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		providers: map[models.AuthMethod]AuthProvider{
			models.AuthMethodPassword:   &PasswordAuthProvider{},
			models.AuthMethodPrivateKey: NewPrivateKeyAuthProvider(),
			models.AuthMethodAgent:      &AgentAuthProvider{},
		},
	}
}

// AddProvider adds a custom authentication provider
func (am *AuthManager) AddProvider(method models.AuthMethod, provider AuthProvider) {
	am.providers[method] = provider
}

// GetAuthMethods returns authentication methods for the given configuration
func (am *AuthManager) GetAuthMethods(config models.SSHConnectionConfig) ([]ssh.AuthMethod, error) {
	provider, exists := am.providers[config.AuthMethod]
	if !exists {
		return nil, fmt.Errorf("unsupported authentication method: %s", config.AuthMethod)
	}
	
	return provider.GetAuthMethods(config)
}

// GetAllAuthMethods tries all available authentication methods
func (am *AuthManager) GetAllAuthMethods(config models.SSHConnectionConfig) []ssh.AuthMethod {
	var allMethods []ssh.AuthMethod
	
	// Try authentication methods in order of preference
	authOrder := []models.AuthMethod{
		models.AuthMethodAgent,      // SSH agent (if available)
		models.AuthMethodPrivateKey, // Private key
		models.AuthMethodPassword,   // Password (least secure)
	}
	
	for _, method := range authOrder {
		tempConfig := config
		tempConfig.AuthMethod = method
		
		methods, err := am.GetAuthMethods(tempConfig)
		if err == nil {
			allMethods = append(allMethods, methods...)
		}
	}
	
	return allMethods
}

// HostKeyCallback creates a host key callback function
func (am *AuthManager) HostKeyCallback(policy string, knownHostsFile string) (ssh.HostKeyCallback, error) {
	switch strings.ToLower(policy) {
	case "strict":
		if knownHostsFile == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			knownHostsFile = filepath.Join(homeDir, ".ssh", "known_hosts")
		}
		
		callback, err := knownhosts.New(knownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load known hosts: %w", err)
		}
		return callback, nil
		
	case "accept":
		// Accept all host keys (insecure)
		return ssh.InsecureIgnoreHostKey(), nil
		
	case "ask":
		// Interactive host key verification
		return am.createInteractiveHostKeyCallback(knownHostsFile), nil
		
	default:
		return nil, fmt.Errorf("unknown host key policy: %s", policy)
	}
}

// createInteractiveHostKeyCallback creates an interactive host key callback
func (am *AuthManager) createInteractiveHostKeyCallback(knownHostsFile string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// In a real implementation, this would prompt the user
		// For now, we'll accept and save the key
		fingerprint := ssh.FingerprintSHA256(key)
		
		// TODO: Implement interactive prompt
		// For now, automatically accept and save
		if knownHostsFile != "" {
			return am.saveHostKey(knownHostsFile, hostname, key)
		}
		
		// Log the fingerprint for security
		fmt.Printf("Warning: Accepting host key for %s: %s\n", hostname, fingerprint)
		return nil
	}
}

// saveHostKey saves a host key to the known_hosts file
func (am *AuthManager) saveHostKey(knownHostsFile, hostname string, key ssh.PublicKey) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(knownHostsFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Format the host key entry
	keyLine := fmt.Sprintf("%s %s\n", hostname, string(ssh.MarshalAuthorizedKey(key)))
	
	// Append to known_hosts file
	file, err := os.OpenFile(knownHostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()
	
	if _, err := file.WriteString(keyLine); err != nil {
		return fmt.Errorf("failed to write host key: %w", err)
	}
	
	return nil
}

// ValidateConfig validates SSH connection configuration
func (am *AuthManager) ValidateConfig(config models.SSHConnectionConfig) error {
	if config.Host == "" {
		return fmt.Errorf("host is required")
	}
	
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}
	
	if config.Username == "" {
		return fmt.Errorf("username is required")
	}
	
	switch config.AuthMethod {
	case models.AuthMethodPassword:
		if config.Password == "" {
			return fmt.Errorf("password is required for password authentication")
		}
	case models.AuthMethodPrivateKey:
		if config.PrivateKeyPath == "" && len(config.PrivateKeyData) == 0 {
			// Will try to find default keys
		}
	case models.AuthMethodAgent:
		if os.Getenv("SSH_AUTH_SOCK") == "" {
			return fmt.Errorf("SSH agent not available")
		}
	default:
		return fmt.Errorf("unsupported authentication method: %s", config.AuthMethod)
	}
	
	return nil
}
