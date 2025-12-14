package fiber

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/logger"
)

// Config implements fiber middleware struct.
type Config struct {
	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool

	// Config of the logger.
	Config logger.Log

	// CacheControlError max-age caching on chain errors.
	CacheControlError string

	// CheckAliveURI for disabling logging of check alive http calls.
	CheckAliveURI string
}

// ConfigDefault is the default config for fiber.
var ConfigDefault = Config{
	Next:              nil,
	CacheControlError: "max-age=0",
}

func configDefault(config ...Config) Config {
	if len(config) < 1 {
		return ConfigDefault
	}

	cfg := config[0]

	if cfg.Next == nil {
		cfg.Next = ConfigDefault.Next
	}

	return cfg
}

// New creates a new fiber access logging middleware using zerolog.
func New(config ...Config) fiber.Handler {
	var (
		writers    []io.Writer
		cfg        = configDefault(config...)
		start      time.Time
		once       sync.Once
		errHandler fiber.ErrorHandler
	)

	// if cfg.Config.Log.File.Enabled {
	if cfg.Config.File.Enabled {
		writers = append(writers, newRollingAccessFile(&cfg.Config))
	}

	// if Console Log is general enabled and if cfg.Config.Log.EnableAccessLogToConsole is enabled.
	if cfg.Config.Console.Enabled && cfg.Config.EnableAccessLogToConsole {
		if cfg.Config.Console.UseConsoleWriter {
			writers = append(writers, zerolog.ConsoleWriter{
				Out:          os.Stdout,
				NoColor:      false,
				TimeFormat:   zerolog.TimeFieldFormat,
				PartsExclude: []string{"level"},
			})
		} else {
			writers = append(writers, os.Stdout)
		}
	}

	fiberLogger := zerolog.New(
		zerolog.MultiLevelWriter(writers...)).
		With().
		Timestamp().
		Logger().
		Level(zerolog.NoLevel)

	return func(ctx *fiber.Ctx) (err error) {
		// Don't execute middleware if Next returns true
		if cfg.Next != nil && cfg.Next(ctx) {
			return ctx.Next()
		}

		// set error handler once
		once.Do(func() {
			errHandler = ctx.App().ErrorHandler
		})

		start = time.Now()
		// Handle request, store err for logging
		chainErr := ctx.Next()
		if chainErr != nil {
			if errH := errHandler(ctx, chainErr); errH != nil {
				// set HTTP/1.1 500 Internal Server Error
				_ = ctx.SendStatus(fiber.StatusInternalServerError) //nolint:errcheck // ok here
				// ensure also 500 has a Cache-Control
				ctx.Response().Header.Set(fiber.HeaderCacheControl, cfg.CacheControlError)
			}
		}

		elapsed := time.Since(start).Seconds()
		ctx.Locals("elapsed", elapsed)

		// Add performance header
		ctx.Response().Header.Set("X-Performance", fmt.Sprintf("%f", elapsed))

		URI := ctx.Request().RequestURI()
		// do not log checkalive URI
		if cfg.Config.DisableCheckAlive && bytes.Equal(URI, []byte(cfg.CheckAliveURI)) {
			return nil
		}

		// Important note:
		// fiber uses fasthttp to normalize urls.
		// for example a url path like /2//test/2 will be normalized to /2/test/2
		// But for logging we need the unchanged url.
		p := ctx.Path()             // only unchanged path info...
		if len(ctx.Queries()) > 0 { // check if queries are around...
			p = p + "?" + string(ctx.Request().URI().QueryString()) // add query string to request path.
		}

		loggerContext := fiberLogger.Log().Str("IP", ctx.IP()).
			Int("status", ctx.Response().StatusCode()).
			Float64("X-Performance", elapsed).
			Str("URI", p).
			Str("method", ctx.Method()).
			Bytes("host", ctx.Request().Host()).
			Str(fiber.HeaderXForwardedFor, ctx.Get(fiber.HeaderXForwardedFor)).
			Str(fiber.HeaderUserAgent, ctx.Get(fiber.HeaderUserAgent)).
			Str(fiber.HeaderOrigin, ctx.Get(fiber.HeaderOrigin)).
			Str(fiber.HeaderReferer, ctx.Get(fiber.HeaderReferer))

		// error to log context
		if chainErr != nil {
			loggerContext.Err(chainErr)
		}

		// send content
		loggerContext.Send()

		// end chain
		return nil
	}
}

// newRollingAccessFile uses lumberjack to create file based access log.
func newRollingAccessFile(cfg *logger.Log) io.Writer {
	// create log folder if defined.
	if cfg.File.Path != "" {
		if err := os.MkdirAll(cfg.File.Path, 0o750); err != nil {
			log.Error().Err(err).Str("path", cfg.File.Path).Msg("can't create log directory")

			return nil
		}
	}

	return &lumberjack.Logger{
		Filename:   path.Join(cfg.File.Path, cfg.File.AccessLog),
		MaxSize:    cfg.File.AccessMaxSize,
		MaxAge:     cfg.File.AccessMaxAge,
		MaxBackups: cfg.File.AccessMaxBackups,
		LocalTime:  false,
		Compress:   false,
	}
}
