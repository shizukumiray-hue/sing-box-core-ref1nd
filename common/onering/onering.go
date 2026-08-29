package onering

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	E "github.com/sagernet/sing/common/exceptions"
)

// Prefix for onering format
const (
	Prefix = "onering:"
	// BUG FIX #4: Max input length to prevent DoS (1KB is enough for domain strings)
	MaxInputLength = 1024
)

// BUG FIX #2: Domain validation regex - allows alphanumeric, dots, hyphens, and optional port
// Format: [subdomain.]domain.tld[:port]
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(:[0-9]{1,5})?$`)

// Config holds parsed onering configuration
type Config struct {
	// BUG FIX #5: Add mutex for thread-safe access
	mu         sync.RWMutex
	Enabled    bool
	RealDomain string
	BugDomain  string
}

// Parse parses onering format string
// Format: "onering:real_domain:bug_domain"
// Returns Config or error if invalid format
func Parse(input string) (*Config, error) {
	// BUG FIX #4: Check input length to prevent DoS
	if len(input) > MaxInputLength {
		return nil, E.New("input too long: maximum ", MaxInputLength, " bytes allowed")
	}

	// Empty input = disabled
	if input == "" {
		return &Config{Enabled: false}, nil
	}

	// Check for onering format
	if !strings.HasPrefix(input, Prefix) {
		// Not onering format = disabled (backward compatible)
		return &Config{
			Enabled:    false,
			RealDomain: input,
			BugDomain:  "",
		}, nil
	}

	// Remove prefix
	trimmed := strings.TrimPrefix(input, Prefix)

	// Split by ":"
	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return nil, E.New("invalid onering format: expected 'onering:real:bug'")
	}

	// Check for invalid chars in raw parts BEFORE trimming
	if containsInvalidChars(parts[0]) || containsInvalidChars(parts[1]) {
		return nil, E.New("domain contains invalid characters")
	}

	// Trim spaces from each part
	real := strings.TrimSpace(parts[0])
	bug := strings.TrimSpace(parts[1])

	// Validation after trim
	if real == "" {
		return nil, E.New("real domain cannot be empty")
	}
	if bug == "" {
		return nil, E.New("bug domain cannot be empty")
	}

	// BUG FIX #2: Validate domain format using regex
	if !isValidDomain(real) {
		return nil, E.New("invalid real domain format: ", real)
	}
	if !isValidDomain(bug) {
		return nil, E.New("invalid bug domain format: ", bug)
	}

	return &Config{
		Enabled:    true,
		RealDomain: real,
		BugDomain:  bug,
	}, nil
}

// containsInvalidChars checks for characters that shouldn't be in domain names
func containsInvalidChars(domain string) bool {
	// Reject control characters (but allow space 32 since it will be trimmed)
	for _, r := range domain {
		if r < 32 || r == 127 {
			return true
		}
	}
	// Reject common injection patterns (but not space)
	return strings.ContainsAny(domain, "\r\n\t\"'<>")
}

// BUG FIX #2: isValidDomain validates domain format using regex
// Allows: alphanumeric, dots, hyphens, and optional port
// Examples: "example.com", "sub.example.com", "example.com:443"
// Rejects: "my domain.com", "domain..com", "-domain.com", "domain-.com"
// NOTE: IPv6 addresses are not supported. Use hostnames only.
func isValidDomain(domain string) bool {
	// Empty check
	if domain == "" {
		return false
	}
	// Length check (max domain length is 253 characters)
	if len(domain) > 253 {
		return false
	}
	// Check with regex
	if !domainRegex.MatchString(domain) {
		return false
	}
	
	// REVIEWER FIX #1: Validate port range if present
	// Regex accepts [0-9]{1,5} which allows invalid ports like 99999 or 65536
	// We need semantic validation to ensure port is in valid range 1-65535
	if portIdx := strings.LastIndex(domain, ":"); portIdx != -1 {
		portStr := domain[portIdx+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return false
		}
	}
	
	return true
}

// GetDialAddress returns the address to dial (bug domain if enabled)
// BUG FIX #5: Thread-safe access with RWMutex
func (c *Config) GetDialAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Enabled && c.BugDomain != "" {
		return c.BugDomain
	}
	return c.RealDomain
}

// GetTLSSNI returns SNI for TLS handshake
// BUG FIX #5: Thread-safe access with RWMutex
func (c *Config) GetTLSSNI() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Enabled && c.BugDomain != "" {
		return c.BugDomain
	}
	return c.RealDomain
}

// GetHTTPHost returns Host header for HTTP/WebSocket
// BUG FIX #5: Thread-safe access with RWMutex
func (c *Config) GetHTTPHost() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RealDomain
}

// String returns human-readable format
// BUG FIX #5: Thread-safe access with RWMutex
func (c *Config) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.Enabled {
		return "onering:disabled"
	}
	return "onering:enabled(real=" + c.RealDomain + ",bug=" + c.BugDomain + ")"
}
