package onering

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		enabled   bool
		realDomain string
		bugDomain  string
	}{
		{
			name:       "valid format",
			input:      "onering:real.com:bug.com",
			wantErr:    false,
			enabled:    true,
			realDomain: "real.com",
			bugDomain:  "bug.com",
		},
		{
			name:       "valid with spaces",
			input:      "onering: real.com : bug.com ",
			wantErr:    false,
			enabled:    true,
			realDomain: "real.com",
			bugDomain:  "bug.com",
		},
		{
			name:    "invalid format - missing bug",
			input:   "onering:real.com",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			input:   "onering:real.com:bug.com:extra",
			wantErr: true,
		},
		{
			name:    "invalid format - empty real",
			input:   "onering::bug.com",
			wantErr: true,
		},
		{
			name:    "invalid format - empty bug",
			input:   "onering:real.com:",
			wantErr: true,
		},
		{
			name:       "empty input",
			input:      "",
			wantErr:    false,
			enabled:    false,
			realDomain: "",
			bugDomain:  "",
		},
		{
			name:       "non-onering format (backward compatible)",
			input:      "example.com",
			wantErr:    false,
			enabled:    false,
			realDomain: "example.com",
			bugDomain:  "",
		},
		{
			name:    "invalid chars - newline",
			input:   "onering:real.com:bug\n.com",
			wantErr: true,
		},
		{
			name:    "invalid chars - tab",
			input:   "onering:real\t.com:bug.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if cfg.Enabled != tt.enabled {
				t.Errorf("Parse() enabled = %v, want %v", cfg.Enabled, tt.enabled)
			}
			if cfg.RealDomain != tt.realDomain {
				t.Errorf("Parse() realDomain = %v, want %v", cfg.RealDomain, tt.realDomain)
			}
			if cfg.BugDomain != tt.bugDomain {
				t.Errorf("Parse() bugDomain = %v, want %v", cfg.BugDomain, tt.bugDomain)
			}
		})
	}
}

func TestConfig_GetDialAddress(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name: "enabled with bug domain",
			config: &Config{
				Enabled:    true,
				RealDomain: "real.com",
				BugDomain:  "bug.com",
			},
			want: "bug.com",
		},
		{
			name: "disabled",
			config: &Config{
				Enabled:    false,
				RealDomain: "real.com",
				BugDomain:  "",
			},
			want: "real.com",
		},
		{
			name: "enabled but no bug domain",
			config: &Config{
				Enabled:    true,
				RealDomain: "real.com",
				BugDomain:  "",
			},
			want: "real.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetDialAddress(); got != tt.want {
				t.Errorf("Config.GetDialAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetTLSSNI(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name: "enabled with bug domain",
			config: &Config{
				Enabled:    true,
				RealDomain: "real.com",
				BugDomain:  "bug.com",
			},
			want: "bug.com",
		},
		{
			name: "disabled",
			config: &Config{
				Enabled:    false,
				RealDomain: "real.com",
				BugDomain:  "",
			},
			want: "real.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetTLSSNI(); got != tt.want {
				t.Errorf("Config.GetTLSSNI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetHTTPHost(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name: "enabled - should return real domain",
			config: &Config{
				Enabled:    true,
				RealDomain: "real.com",
				BugDomain:  "bug.com",
			},
			want: "real.com",
		},
		{
			name: "disabled - should return real domain",
			config: &Config{
				Enabled:    false,
				RealDomain: "real.com",
				BugDomain:  "",
			},
			want: "real.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetHTTPHost(); got != tt.want {
				t.Errorf("Config.GetHTTPHost() = %v, want %v", got, tt.want)
			}
		})
	}
}
