package logger

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Package logger implements feed main logger.

// LevelWriter implements a struct to split logs by info and error and up level.
// See func WriteLevel about the separation.
type LevelWriter struct {
	io.Writer
	ErrorWriter io.Writer
	InfoWriter  io.Writer
	TraceWriter io.Writer
	WarnWriter  io.Writer
}

// WriteLevel splits logging by level and links the pointer to the target output depending on the logger defined.
func (lw *LevelWriter) WriteLevel(l zerolog.Level, p []byte) (n int, err error) {
	var w io.Writer

	// disabled logging
	if l == zerolog.Disabled {
		return 0, nil
	}

	// decide where to write this log content
	switch {
	case l == zerolog.TraceLevel:
		w = lw.TraceWriter
	case l == zerolog.WarnLevel:
		w = lw.WarnWriter
	case l > zerolog.WarnLevel: // error and fatal panic go to error
		w = lw.ErrorWriter
	default:
		w = lw.InfoWriter // debug and info go to info
	}

	// return selected logger writer.
	return w.Write(p) //nolint:wrapcheck
}

// Init the zerolog logger.
// Depending on the config it enables all, some or no logger at all.
// Be sure to enable at least one logger for output.
func Init(cfg Log) error { //nolint:funlen
	var (
		logLevel, err = zerolog.ParseLevel(cfg.LogLevel)
		writers       []io.Writer
		stack         bool
	)

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("loglevel %s is not supported", cfg.LogLevel))
	}

	if cfg.ServiceName == "" {
		return ErrServiceNameIsEmpty
	}

	if cfg.AppName == "" {
		return ErrAppNameIsEmpty
	}

	// use zerolog stack marshal func if trace level is set
	if logLevel == zerolog.TraceLevel {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack //nolint:reassign
		stack = true
	}

	zerolog.SetGlobalLevel(logLevel)

	// init prometheus
	ph := NewPrometheusHook(cfg.ServiceName)

	// add the enabled only loggers
	if cfg.Console.Enabled {
		writers = append(writers, NewConsoleWriter(cfg))
	}

	if cfg.File.Enabled {
		writers = append(writers, newRollingInfoErrorFile(cfg))
	}

	mw := zerolog.MultiLevelWriter(writers...)

	// decide what zero log should show
	switch {
	case cfg.ReportCaller && stack:
		log.Logger = zerolog.New(mw).Hook(ph).With().Timestamp().Stack().Logger()
	case cfg.ReportCaller:
		log.Logger = zerolog.New(mw).Hook(ph).With().Timestamp().Caller().Logger()
	default:
		log.Logger = zerolog.New(mw).Hook(ph).With().Timestamp().Logger()
	}

	return nil
}

// newRollingInfoErrorFile uses LevelWriter and lumberjack to create file based log.
func newRollingInfoErrorFile(cfg Log) io.Writer {
	if err := os.MkdirAll(cfg.File.Path, 0o750); err != nil { //nolint: mnd
		log.Error().Err(err).Str("path", cfg.File.Path).Msg("can't create log directory")

		return nil
	}

	var lw LevelWriter

	lw.ErrorWriter = &lumberjack.Logger{
		Filename:   path.Join(cfg.File.Path, cfg.File.ErrorLog),
		MaxSize:    cfg.File.ErrorMaxSize,
		MaxAge:     cfg.File.ErrorMaxAge,
		MaxBackups: cfg.File.ErrorMaxBackups,
		LocalTime:  false,
		Compress:   false,
	}

	lw.InfoWriter = &lumberjack.Logger{
		Filename:   path.Join(cfg.File.Path, cfg.File.InfoLog),
		MaxSize:    cfg.File.InfoMaxSize,
		MaxAge:     cfg.File.InfoMaxAge,
		MaxBackups: cfg.File.InfoMaxBackups,
		LocalTime:  false,
		Compress:   false,
	}

	lw.TraceWriter = &lumberjack.Logger{
		Filename:   path.Join(cfg.File.Path, cfg.File.TraceLog),
		MaxSize:    cfg.File.TraceMaxSize,
		MaxAge:     cfg.File.TraceMaxAge,
		MaxBackups: cfg.File.TraceMaxBackups,
		LocalTime:  false,
		Compress:   false,
	}

	lw.WarnWriter = &lumberjack.Logger{
		Filename:   path.Join(cfg.File.Path, cfg.File.WarnLog),
		MaxSize:    cfg.File.WarnMaxSize,
		MaxAge:     cfg.File.WarnMaxAge,
		MaxBackups: cfg.File.WarnMaxBackups,
		LocalTime:  false,
		Compress:   false,
	}

	return &lw
}

// NewConsoleWriter creates a zerolog ConsoleWriter.
func NewConsoleWriter(cfg Log) io.Writer {
	var lw LevelWriter

	lw.ErrorWriter = os.Stderr
	lw.InfoWriter = os.Stdout
	lw.TraceWriter = os.Stderr
	lw.WarnWriter = os.Stderr

	if cfg.Console.UseConsoleWriter {
		lw.ErrorWriter = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			NoColor:    false,
			TimeFormat: zerolog.TimeFieldFormat,
		}

		lw.InfoWriter = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			NoColor:    false,
			TimeFormat: zerolog.TimeFieldFormat,
		}

		lw.TraceWriter = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			NoColor:    false,
			TimeFormat: zerolog.TimeFieldFormat,
		}

		lw.WarnWriter = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			NoColor:    false,
			TimeFormat: zerolog.TimeFieldFormat,
		}
	}

	return &lw
}
