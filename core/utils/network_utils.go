package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// NetworkUtils provides network-related utility functions
type NetworkUtils struct{}

// NewNetworkUtils creates a new NetworkUtils instance
func NewNetworkUtils() *NetworkUtils {
	return &NetworkUtils{}
}

// IsPortAvailable checks if a port is available for binding on the given address
func (nu *NetworkUtils) IsPortAvailable(host string, port int) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

// FindAvailablePort finds an available port starting from the given port
func (nu *NetworkUtils) FindAvailablePort(host string, startPort int) (int, error) {
	for port := startPort; port <= 65535; port++ {
		if nu.IsPortAvailable(host, port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found starting from %d", startPort)
}

// ValidateIPAddress validates if the given string is a valid IP address
func (nu *NetworkUtils) ValidateIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidatePort validates if the given port number is valid
func (nu *NetworkUtils) ValidatePort(port int) bool {
	return port > 0 && port <= 65535
}

// ValidateHostPort validates host:port format
func (nu *NetworkUtils) ValidateHostPort(hostport string) error {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return fmt.Errorf("invalid host:port format: %w", err)
	}
	
	// Validate host
	if host != "" && !nu.ValidateIPAddress(host) {
		// Check if it's a valid hostname (basic check)
		if strings.Contains(host, " ") || strings.Contains(host, "/") {
			return fmt.Errorf("invalid host: %s", host)
		}
	}
	
	// Validate port
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s", portStr)
	}
	
	if !nu.ValidatePort(port) {
		return fmt.Errorf("port out of range: %d", port)
	}
	
	return nil
}

// TestConnection tests if a connection can be established to the given address
func (nu *NetworkUtils) TestConnection(address string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer conn.Close()
	return nil
}

// GetLocalIP gets the local IP address
func (nu *NetworkUtils) GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to get local IP: %w", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// GetInterfaceIPs returns all IP addresses of network interfaces
func (nu *NetworkUtils) GetInterfaceIPs() ([]string, error) {
	var ips []string
	
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}
	
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			
			if ip == nil || ip.IsLoopback() {
				continue
			}
			
			ip = ip.To4()
			if ip == nil {
				continue // not an IPv4 address
			}
			
			ips = append(ips, ip.String())
		}
	}
	
	return ips, nil
}

// ParseBindAddress parses bind address and returns host and port
func (nu *NetworkUtils) ParseBindAddress(bind string, defaultPort int) (string, int, error) {
	if bind == "" {
		return "127.0.0.1", defaultPort, nil
	}
	
	// Check if it's just a port number
	if port, err := strconv.Atoi(bind); err == nil {
		if !nu.ValidatePort(port) {
			return "", 0, fmt.Errorf("invalid port: %d", port)
		}
		return "127.0.0.1", port, nil
	}
	
	// Parse host:port
	host, portStr, err := net.SplitHostPort(bind)
	if err != nil {
		// Maybe it's just a host without port
		if nu.ValidateIPAddress(bind) || bind == "localhost" {
			return bind, defaultPort, nil
		}
		return "", 0, fmt.Errorf("invalid bind address: %s", bind)
	}
	
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", portStr)
	}
	
	if !nu.ValidatePort(port) {
		return "", 0, fmt.Errorf("port out of range: %d", port)
	}
	
	if host == "" {
		host = "127.0.0.1"
	}
	
	return host, port, nil
}

// ResolveAddress resolves hostname to IP address
func (nu *NetworkUtils) ResolveAddress(hostname string) ([]string, error) {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", hostname, err)
	}
	
	var addresses []string
	for _, ip := range ips {
		addresses = append(addresses, ip.String())
	}
	
	return addresses, nil
}

// IsLocalAddress checks if the given address is a local address
func (nu *NetworkUtils) IsLocalAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	
	return ip.IsLoopback() || ip.IsUnspecified()
}
