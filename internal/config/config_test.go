package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadConfig(t *testing.T) {
	var (
		err         error
		projectRoot string
	)

	// Get the project root by going up from internal/config
	projectRoot, err = filepath.Abs("../../")
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	configPath := filepath.Join(projectRoot, "etc") + string(filepath.Separator)

	var cfg Config

	cfg, err = ReadConfig(configPath)
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	// Test basic config fields
	if cfg.Title == "" {
		t.Error("Config.Title should not be empty")
	}

	if cfg.Webserver.Port == 0 {
		t.Error("Webserver.Port should not be 0")
	}

	if cfg.Webserver.URL == "" {
		t.Error("Webserver.URL should not be empty")
	}

	// Test DB config
	if cfg.DB.GormEngine == "" {
		t.Error("DB.GormEngine should not be empty")
	}
}

// validBase returns a Config that passes all validation rules.
func validBase() Config {
	return Config{
		Webserver: Webserver{
			Port:                8080,
			URL:                 "http://localhost:8080",
			CookieEncryptionKey: "a-random-string-that-is-at-least-32-chars!",
			Argon2Salt:          "a-random-salt-16c!",
		},
		DB: DB{GormEngine: "sqlite"},
		Auth: Auth{
			LocalDB: LocalDBAuth{Enabled: true},
		},
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name:    "valid config",
			config:  validBase(),
			wantErr: nil,
		},
		{
			name: "missing port",
			config: func() Config {
				c := validBase()
				c.Webserver.Port = 0

				return c
			}(),
			wantErr: ErrWebServerPortCanNotBeZero,
		},
		{
			name: "missing URL",
			config: func() Config {
				c := validBase()
				c.Webserver.URL = ""

				return c
			}(),
			wantErr: ErrEmptyURL,
		},
		{
			name: "placeholder cookie key",
			config: func() Config {
				c := validBase()
				c.Webserver.CookieEncryptionKey = placeholderSecret

				return c
			}(),
			wantErr: ErrPlaceholderCookieKey,
		},
		{
			name: "cookie key too short",
			config: func() Config {
				c := validBase()
				c.Webserver.CookieEncryptionKey = "tooshort"

				return c
			}(),
			wantErr: ErrCookieKeyTooShort,
		},
		{
			name: "placeholder argon2 salt",
			config: func() Config {
				c := validBase()
				c.Webserver.Argon2Salt = placeholderSecret

				return c
			}(),
			wantErr: ErrPlaceholderArgon2Salt,
		},
		{
			name: "argon2 salt too short",
			config: func() Config {
				c := validBase()
				c.Webserver.Argon2Salt = "short"

				return c
			}(),
			wantErr: ErrArgon2SaltTooShort,
		},
		{
			name: "missing DB engine",
			config: func() Config {
				c := validBase()
				c.DB.GormEngine = ""

				return c
			}(),
			wantErr: ErrDBMissingEngine,
		},
		{
			name: "no auth provider enabled",
			config: func() Config {
				c := validBase()
				c.Auth.LocalDB.Enabled = false

				return c
			}(),
			wantErr: ErrNoAuthProviderEnabled,
		},
		{
			name: "OIDC enabled missing provider URL",
			config: func() Config {
				c := validBase()
				c.Auth.OIDC = OIDCAuth{Enabled: true, ClientID: "id", ClientSecret: "secret", RedirectURL: "http://x"}

				return c
			}(),
			wantErr: ErrOIDCMissingProviderURL,
		},
		{
			name: "OIDC enabled missing client ID",
			config: func() Config {
				c := validBase()
				c.Auth.OIDC = OIDCAuth{Enabled: true, ProviderURL: "https://x", ClientSecret: "secret", RedirectURL: "http://x"}

				return c
			}(),
			wantErr: ErrOIDCMissingClientID,
		},
		{
			name: "OIDC enabled missing client secret",
			config: func() Config {
				c := validBase()
				c.Auth.OIDC = OIDCAuth{Enabled: true, ProviderURL: "https://x", ClientID: "id", RedirectURL: "http://x"}

				return c
			}(),
			wantErr: ErrOIDCMissingClientSecret,
		},
		{
			name: "OIDC enabled missing redirect URL",
			config: func() Config {
				c := validBase()
				c.Auth.OIDC = OIDCAuth{Enabled: true, ProviderURL: "https://x", ClientID: "id", ClientSecret: "secret"}

				return c
			}(),
			wantErr: ErrOIDCMissingRedirectURL,
		},
		{
			name: "LDAP enabled missing host",
			config: func() Config {
				c := validBase()
				c.Auth.LDAP = LDAPAuth{Enabled: true, Port: 389, BaseDN: "dc=example,dc=com"}

				return c
			}(),
			wantErr: ErrLDAPMissingHost,
		},
		{
			name: "LDAP enabled missing port",
			config: func() Config {
				c := validBase()
				c.Auth.LDAP = LDAPAuth{Enabled: true, Host: "ldap.example.com", BaseDN: "dc=example,dc=com"}

				return c
			}(),
			wantErr: ErrLDAPMissingPort,
		},
		{
			name: "LDAP enabled missing base DN",
			config: func() Config {
				c := validBase()
				c.Auth.LDAP = LDAPAuth{Enabled: true, Host: "ldap.example.com", Port: 389}

				return c
			}(),
			wantErr: ErrLDAPMissingBaseDN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(&tt.config)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validate() unexpected error: %v", err)
				}

				return
			}

			if err == nil {
				t.Errorf("validate() expected error %v, got nil", tt.wantErr)
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadConfigWithOverlayFile(t *testing.T) {
	projectRoot, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	// Write a temporary overlay that changes only the Title.
	overlayDir := filepath.Join(projectRoot, "etc", "local")
	if err = os.MkdirAll(overlayDir, 0o750); err != nil {
		t.Fatalf("failed to create overlay dir: %v", err)
	}

	overlayFile := filepath.Join(overlayDir, "test-overlay.toml")
	if err = os.WriteFile(overlayFile, []byte(`title = "Overlay Title"`), 0o600); err != nil {
		t.Fatalf("failed to write overlay file: %v", err)
	}

	t.Cleanup(func() { _ = os.Remove(overlayFile) })

	cfg, err := ReadConfig(overlayFile)
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if cfg.Title != "Overlay Title" {
		t.Errorf("Title = %q, want %q", cfg.Title, "Overlay Title")
	}

	// Fields not in the overlay must still come from main.toml.
	if cfg.Webserver.Port == 0 {
		t.Error("Webserver.Port should be set from main.toml")
	}
}

func TestReadConfigWithEnvOverride(t *testing.T) {
	projectRoot, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	configPath := filepath.Join(projectRoot, "etc") + string(filepath.Separator)

	// Viper env vars: prefix GPDNS_ + uppercase key path with _ separator.
	t.Setenv("GPDNS_TITLE", "Test Override")
	t.Setenv("GPDNS_WEBSERVER_PORT", "9090")

	cfg, err := ReadConfig(configPath)
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if cfg.Title != "Test Override" {
		t.Errorf("Title = %v, want %v", cfg.Title, "Test Override")
	}

	if cfg.Webserver.Port != 9090 {
		t.Errorf("Webserver.Port = %v, want %v", cfg.Webserver.Port, 9090)
	}
}


func TestDumpConfigJSON(t *testing.T) {
	var err error

	cfg := Config{
		Title:   "Test",
		DevMode: true,
		Webserver: Webserver{
			Port: 8080,
			URL:  "http://localhost:8080",
		},
	}

	var jsonStr string

	jsonStr, err = DumpConfigJSON(&cfg)
	if err != nil {
		t.Fatalf("DumpConfigJSON() error = %v", err)
	}

	if jsonStr == "" {
		t.Error("DumpConfigJSON() returned empty string")
	}

	// Check if output is valid JSON by checking for expected fields
	if !strings.Contains(jsonStr, "Test") {
		t.Error("DumpConfigJSON() output should contain Title")
	}
}
