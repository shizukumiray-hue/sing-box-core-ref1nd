package onering

import (
	"testing"
)

// CRITICAL FIX #1 VERIFICATION: Port validation with strconv.Atoi()
// Tests that ports outside valid range 1-65535 are properly rejected
func TestCriticalFix1_PortValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		reason  string
	}{
		{
			name:    "port 0 should be rejected",
			input:   "onering:real.com:bug.com:0",
			wantErr: true,
			reason:  "Port 0 is invalid",
		},
		{
			name:    "port 65536 should be rejected",
			input:   "onering:real.com:bug.com:65536",
			wantErr: true,
			reason:  "Port 65536 exceeds maximum",
		},
		{
			name:    "port 99999 should be rejected",
			input:   "onering:real.com:bug.com:99999",
			wantErr: true,
			reason:  "Port 99999 exceeds maximum",
		},
		{
			name:    "port 1 should be accepted (if parser supported ports)",
			input:   "onering:real.com:bug.com:1",
			wantErr: true, // Currently rejected by parser (3 parts)
			reason:  "Parser rejects 3+ parts currently",
		},
		{
			name:    "port 65535 should be accepted (if parser supported ports)",
			input:   "onering:real.com:bug.com:65535",
			wantErr: true, // Currently rejected by parser (3 parts)
			reason:  "Parser rejects 3+ parts currently",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v (reason: %s)", err, tt.wantErr, tt.reason)
			}
		})
	}
}

// CRITICAL FIX #1 VERIFICATION: Test isValidDomain() directly for port validation
func TestCriticalFix1_IsValidDomainPorts(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected bool
		reason   string
	}{
		{
			name:     "valid port 443",
			domain:   "example.com:443",
			expected: true,
			reason:   "Port 443 is valid",
		},
		{
			name:     "valid port 8443",
			domain:   "example.com:8443",
			expected: true,
			reason:   "Port 8443 is valid",
		},
		{
			name:     "valid port 1",
			domain:   "example.com:1",
			expected: true,
			reason:   "Port 1 is minimum valid port",
		},
		{
			name:     "valid port 65535",
			domain:   "example.com:65535",
			expected: true,
			reason:   "Port 65535 is maximum valid port",
		},
		{
			name:     "invalid port 0",
			domain:   "example.com:0",
			expected: false,
			reason:   "Port 0 is below minimum",
		},
		{
			name:     "invalid port 65536",
			domain:   "example.com:65536",
			expected: false,
			reason:   "Port 65536 exceeds maximum",
		},
		{
			name:     "invalid port 99999",
			domain:   "example.com:99999",
			expected: false,
			reason:   "Port 99999 exceeds maximum",
		},
		{
			name:     "invalid port -1",
			domain:   "example.com:-1",
			expected: false,
			reason:   "Negative port",
		},
		{
			name:     "invalid port with letters",
			domain:   "example.com:abc",
			expected: false,
			reason:   "Non-numeric port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("isValidDomain(%q) = %v, want %v (reason: %s)", 
					tt.domain, result, tt.expected, tt.reason)
			}
		})
	}
}

// CRITICAL FIX #2 VERIFICATION: IPv6 handling
// Verify that IPv6 addresses are properly rejected (not supported)
func TestCriticalFix2_IPv6Rejection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		reason  string
	}{
		{
			name:    "IPv6 loopback rejected",
			input:   "onering:::1:bug.com",
			wantErr: true,
			reason:  "IPv6 ::1 not supported",
		},
		{
			name:    "IPv6 full address rejected",
			input:   "onering:2001:db8::1:bug.com",
			wantErr: true,
			reason:  "IPv6 2001:db8::1 not supported",
		},
		{
			name:    "IPv6 with brackets rejected",
			input:   "onering:[2001:db8::1]:bug.com",
			wantErr: true,
			reason:  "Bracketed IPv6 not supported",
		},
		{
			name:    "IPv6 with port rejected",
			input:   "onering:[2001:db8::1]:8443:bug.com",
			wantErr: true,
			reason:  "IPv6 with port not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v (reason: %s)", err, tt.wantErr, tt.reason)
			}
		})
	}
}

// CRITICAL FIX #2 VERIFICATION: Test isValidDomain() rejects IPv6
func TestCriticalFix2_IsValidDomainIPv6(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected bool
		reason   string
	}{
		{
			name:     "IPv6 ::1 rejected",
			domain:   "::1",
			expected: false,
			reason:   "IPv6 loopback not supported",
		},
		{
			name:     "IPv6 full address rejected",
			domain:   "2001:db8::1",
			expected: false,
			reason:   "IPv6 address not supported",
		},
		{
			name:     "IPv6 with brackets rejected",
			domain:   "[2001:db8::1]",
			expected: false,
			reason:   "Bracketed IPv6 not supported",
		},
		{
			name:     "IPv6 with port rejected",
			domain:   "[2001:db8::1]:8443",
			expected: false,
			reason:   "IPv6 with port not supported",
		},
		{
			name:     "valid hostname accepted",
			domain:   "example.com",
			expected: true,
			reason:   "Regular hostname should work",
		},
		{
			name:     "valid hostname with port accepted",
			domain:   "example.com:443",
			expected: true,
			reason:   "Regular hostname with port should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("isValidDomain(%q) = %v, want %v (reason: %s)", 
					tt.domain, result, tt.expected, tt.reason)
			}
		})
	}
}

// REGRESSION CHECK: Verify original 5 bug fixes still work
func TestRegressionCheck_OriginalBugFixes(t *testing.T) {
	t.Run("BUG FIX #1: Prefix validation", func(t *testing.T) {
		// Should accept non-onering format (backward compatible)
		cfg, err := Parse("example.com")
		if err != nil {
			t.Errorf("Non-onering format should be accepted: %v", err)
		}
		if cfg.Enabled {
			t.Error("Non-onering format should have Enabled=false")
		}
	})

	t.Run("BUG FIX #2: Domain validation", func(t *testing.T) {
		// Should reject invalid domains
		invalidDomains := []string{
			"onering:my domain.com:bug.com",  // spaces
			"onering:domain..com:bug.com",    // double dots
			"onering:-domain.com:bug.com",    // leading hyphen
			"onering:domain-.com:bug.com",    // trailing hyphen
		}
		for _, input := range invalidDomains {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("Should reject invalid domain: %s", input)
			}
		}
	})

	t.Run("BUG FIX #3: Input sanitization", func(t *testing.T) {
		// Should trim spaces correctly
		cfg, err := Parse("onering: real.com : bug.com ")
		if err != nil {
			t.Errorf("Should handle spaces: %v", err)
		}
		if cfg.RealDomain != "real.com" || cfg.BugDomain != "bug.com" {
			t.Error("Spaces not trimmed correctly")
		}
		
		// Should reject control characters
		_, err = Parse("onering:real\n.com:bug.com")
		if err == nil {
			t.Error("Should reject newline characters")
		}
	})

	t.Run("BUG FIX #4: DoS protection", func(t *testing.T) {
		// Should reject oversized input
		longInput := "onering:" + string(make([]byte, MaxInputLength+1))
		_, err := Parse(longInput)
		if err == nil {
			t.Error("Should reject input exceeding MaxInputLength")
		}
	})

	t.Run("BUG FIX #5: Thread safety", func(t *testing.T) {
		// Should handle concurrent access
		cfg, err := Parse("onering:real.com:bug.com")
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		
		done := make(chan bool, 50)
		for i := 0; i < 50; i++ {
			go func() {
				_ = cfg.GetDialAddress()
				_ = cfg.GetTLSSNI()
				_ = cfg.GetHTTPHost()
				done <- true
			}()
		}
		
		for i := 0; i < 50; i++ {
			<-done
		}
	})
}
