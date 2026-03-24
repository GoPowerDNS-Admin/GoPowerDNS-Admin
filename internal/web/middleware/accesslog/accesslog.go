// Package accesslog provides a Fiber middleware that logs each HTTP request
// using zerolog at Info level.
package accesslog

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

// New returns a Fiber middleware that logs method, path, status, latency,
// and remote IP for every request.
func New() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		if c.Path() != "/health" {
			log.Info().
				Str("method", c.Method()).
				Str("path", c.Path()).
				Int("status", c.Response().StatusCode()).
				Dur("latency", time.Since(start)).
				Str("ip", c.IP()).
				Msg("request")
		}

		return err
	}
}
