package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	
	"golang.org/x/crypto/ssh"
)

// CryptoUtils provides cryptographic utilities for SSH operations
type CryptoUtils struct{}

// NewCryptoUtils creates a new CryptoUtils instance
func NewCryptoUtils() *CryptoUtils {
	return &CryptoUtils{}
}

// GenerateKeyPair generates a new RSA key pair
func (cu *CryptoUtils) GenerateKeyPair(bits int) ([]byte, []byte, error) {
	if bits < 2048 {
		bits = 2048 // Minimum secure key size
	}
	
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	
	// Encode private key to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)
	
	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate public key: %w", err)
	}
	
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	
	return privateKeyBytes, publicKeyBytes, nil
}

// LoadPrivateKey loads a private key from file or bytes
func (cu *CryptoUtils) LoadPrivateKey(keyData []byte, passphrase string) (ssh.Signer, error) {
	var signer ssh.Signer
	var err error
	
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	
	return signer, nil
}

// LoadPrivateKeyFromFile loads a private key from a file
func (cu *CryptoUtils) LoadPrivateKeyFromFile(keyPath, passphrase string) (ssh.Signer, error) {
	keyData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	
	return cu.LoadPrivateKey(keyData, passphrase)
}

// SavePrivateKey saves a private key to file with proper permissions
func (cu *CryptoUtils) SavePrivateKey(keyData []byte, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write key file with restrictive permissions
	if err := ioutil.WriteFile(filePath, keyData, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	
	return nil
}

// SavePublicKey saves a public key to file
func (cu *CryptoUtils) SavePublicKey(keyData []byte, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write public key file
	if err := ioutil.WriteFile(filePath, keyData, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	
	return nil
}

// ValidatePrivateKey validates a private key
func (cu *CryptoUtils) ValidatePrivateKey(keyData []byte, passphrase string) error {
	_, err := cu.LoadPrivateKey(keyData, passphrase)
	return err
}

// ValidatePrivateKeyFile validates a private key file
func (cu *CryptoUtils) ValidatePrivateKeyFile(keyPath, passphrase string) error {
	_, err := cu.LoadPrivateKeyFromFile(keyPath, passphrase)
	return err
}

// GetKeyFingerprint returns the fingerprint of a public key
func (cu *CryptoUtils) GetKeyFingerprint(publicKey ssh.PublicKey) string {
	return ssh.FingerprintSHA256(publicKey)
}

// GetKeyFingerprintFromFile returns the fingerprint of a public key from file
func (cu *CryptoUtils) GetKeyFingerprintFromFile(keyPath string) (string, error) {
	keyData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read key file: %w", err)
	}
	
	// Try to parse as public key first
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(keyData)
	if err == nil {
		return cu.GetKeyFingerprint(publicKey), nil
	}
	
	// Try to parse as private key
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse key: %w", err)
	}
	
	return cu.GetKeyFingerprint(signer.PublicKey()), nil
}

// EncryptData encrypts data using a simple XOR cipher (for demonstration)
// In production, use proper encryption like AES-GCM
func (cu *CryptoUtils) EncryptData(data []byte, key []byte) []byte {
	if len(key) == 0 {
		return data
	}
	
	encrypted := make([]byte, len(data))
	for i, b := range data {
		encrypted[i] = b ^ key[i%len(key)]
	}
	
	return encrypted
}

// DecryptData decrypts data using a simple XOR cipher (for demonstration)
// In production, use proper encryption like AES-GCM
func (cu *CryptoUtils) DecryptData(encrypted []byte, key []byte) []byte {
	// XOR is symmetric, so decryption is the same as encryption
	return cu.EncryptData(encrypted, key)
}

// GetDefaultKeyPaths returns default SSH key file paths
func (cu *CryptoUtils) GetDefaultKeyPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}
	
	sshDir := filepath.Join(homeDir, ".ssh")
	return []string{
		filepath.Join(sshDir, "id_rsa"),
		filepath.Join(sshDir, "id_ecdsa"),
		filepath.Join(sshDir, "id_ed25519"),
		filepath.Join(sshDir, "id_dsa"),
	}
}

// FindAvailableKeys finds available SSH private keys in default locations
func (cu *CryptoUtils) FindAvailableKeys() []string {
	var availableKeys []string
	
	for _, keyPath := range cu.GetDefaultKeyPaths() {
		if _, err := os.Stat(keyPath); err == nil {
			// Check if key is valid
			if err := cu.ValidatePrivateKeyFile(keyPath, ""); err == nil {
				availableKeys = append(availableKeys, keyPath)
			}
		}
	}
	
	return availableKeys
}
