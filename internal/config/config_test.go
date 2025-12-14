package config

import (
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

	// Test Record map is populated
	if cfg.Record == nil {
		t.Fatal("Record map should not be nil")
	}

	if len(cfg.Record) == 0 {
		t.Error("Record map should not be empty")
	}
}

func TestRecordTypeSettings(t *testing.T) {
	var (
		err         error
		projectRoot string
	)

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

	tests := []struct {
		name            string
		recordType      string
		expectedForward bool
		expectedReverse bool
	}{
		{"A record", "A", true, false},
		{"AAAA record", "AAAA", true, false},
		{"CNAME record", "CNAME", true, false},
		{"MX record", "MX", true, false},
		{"TXT record", "TXT", true, true},
		{"PTR record", "PTR", true, true},
		{"NS record", "NS", true, true},
		{"LOC record", "LOC", true, true},
		{"SOA record", "SOA", false, false},
		{"DNSKEY record", "DNSKEY", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, exists := cfg.Record[tt.recordType]
			if !exists {
				t.Errorf("Record type %s not found in config", tt.recordType)
				return
			}

			if settings.Forward != tt.expectedForward {
				t.Errorf("Record %s Forward = %v, want %v", tt.recordType, settings.Forward, tt.expectedForward)
			}

			if settings.Reverse != tt.expectedReverse {
				t.Errorf("Record %s Reverse = %v, want %v", tt.recordType, settings.Reverse, tt.expectedReverse)
			}
		})
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

func TestReadConfigWithJSONOverride(t *testing.T) {
	var (
		err         error
		projectRoot string
	)

	projectRoot, err = filepath.Abs("../../")
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	configPath := filepath.Join(projectRoot, "etc") + string(filepath.Separator)

	// Set JSON override environment variable
	jsonOverride := `{"Title":"Test Override","Webserver":{"Port":9090}}`
	t.Setenv("GO_POWERDNS_ADMIN_CONFIG_JSON", jsonOverride)

	var cfg Config

	cfg, err = ReadConfig(configPath)
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

func TestDumpConfig(t *testing.T) {
	var err error

	cfg := Config{
		Title:   "Test",
		DevMode: true,
		Webserver: Webserver{
			Port: 8080,
			URL:  "http://localhost:8080",
		},
		Record: Record{
			"A": RecordTypeSettings{
				Forward: true,
				Reverse: false,
			},
		},
	}

	var tomlStr string

	tomlStr, err = DumpConfig(&cfg)
	if err != nil {
		t.Fatalf("DumpConfig() error = %v", err)
	}

	if tomlStr == "" {
		t.Error("DumpConfig() returned empty string")
	}

	// Check if output contains expected values
	if !strings.Contains(tomlStr, "Test") {
		t.Error("DumpConfig() output should contain Title")
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
