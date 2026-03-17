package config

import (
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
	if cfg.DB.Host == "" {
		t.Error("DB.Host should not be empty")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Webserver: Webserver{
					Port: 8080,
					URL:  "http://localhost:8080",
				},
			},
			wantErr: false,
		},
		{
			name: "missing port",
			config: Config{
				Webserver: Webserver{
					Port: 0,
					URL:  "http://localhost:8080",
				},
			},
			wantErr: true,
		},
		{
			name: "missing URL",
			config: Config{
				Webserver: Webserver{
					Port: 8080,
					URL:  "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
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
