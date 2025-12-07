package logger

import (
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// Console implements a console based logger.
type Console struct {
	Enabled          bool `toml:"enabled"`
	UseConsoleWriter bool
}

// LogFile implements a file based logger.
type LogFile struct {
	// Legacy non docker env file logging.
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`

	AccessLog        string `toml:"access"`
	AccessMaxSize    int    `toml:"accessMaxSize"`
	AccessMaxBackups int    `toml:"accessMaxBackups"`
	AccessMaxAge     int    `toml:"accessMaxAge"`

	ErrorLog        string `toml:"error"`
	ErrorMaxSize    int    `toml:"errorMaxSize"`
	ErrorMaxBackups int    `toml:"errorMaxBackups"`
	ErrorMaxAge     int    `toml:"errorMaxAge"`

	InfoLog        string `toml:"info"`
	InfoMaxSize    int    `toml:"infoMaxSize"`
	InfoMaxBackups int    `toml:"infoMaxBackups"`
	InfoMaxAge     int    `toml:"infoMaxAge"`

	TraceLog        string `toml:"trace"`
	TraceMaxSize    int    `toml:"traceMaxSize"`
	TraceMaxBackups int    `toml:"traceMaxBackups"`
	TraceMaxAge     int    `toml:"traceMaxAge"`

	WarnLog        string `toml:"warn"`
	WarnMaxSize    int    `toml:"warnMaxSize"`
	WarnMaxBackups int    `toml:"warnMaxBackups"`
	WarnMaxAge     int    `toml:"warnMaxAge"`
}

// DataDog implements a datadog config.
type DataDog struct {
	ServiceName string                       `toml:"serviceName"`
	APIKey      string                       `toml:"apiKey"` // API Key defined at datadog
	Enabled     bool                         `toml:"enabled"`
	Site        string                       `toml:"site"` // Regional Site aka DD_SITE ("datadoghq.eu")
	SecretName  string                       `toml:"secretname"`
	Servers     datadog.ServerConfigurations `toml:"servers"`
	Timeout     time.Duration                `toml:"timeout"` // how long to wait to send a log entry to datadog.
}

// LogZio implements a logz.io based logger.
type LogZio struct {
	Enabled    bool `toml:"enabled"`
	Debug      bool `toml:"debug"`
	URL        string
	SecretName string
	Token      string
}

// Log implements the logger config.
type Log struct {
	LogLevel string // info, warn, error.
	LogEnv   string

	// EnableAccessLogToConsole if true any feed service having a webservice, will start to log to console.
	// Does not overrule flag Console.Enabled!
	// If Console.Enabled is false, still no access log output to the console will be shown.
	EnableAccessLogToConsole bool
	ReportCaller             bool
	DisableCheckAlive        bool // do not log /checkalive calls

	AppName     string
	ServiceName string

	// Console used mainly for docker and dev.
	Console Console

	// Legacy non docker env file logging.
	File LogFile `toml:"file"`

	// logz.io (used with docker and legacy non dev env).
	LogZio LogZio

	// DataDog
	DataDog DataDog
}
