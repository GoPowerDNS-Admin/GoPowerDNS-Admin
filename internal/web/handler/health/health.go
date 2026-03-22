// Package health provides a health check endpoint for liveness and readiness probes.
package health

import (
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

const (
	// Path is the health check endpoint path.
	Path = "/health"

	dbProbeTimeout = 2 * time.Second
)

// Handler handles health check requests.
type Handler struct {
	db    *gorm.DB
	alive *atomic.Bool
}

// New creates a new health handler. alive is the Service.alive flag that is set
// to false during graceful shutdown, causing the liveness check to fail so that
// a load balancer can drain the instance before it stops.
func New(db *gorm.DB, alive *atomic.Bool) *Handler {
	return &Handler{db: db, alive: alive}
}

// Register registers the health endpoint on the given router. It must be
// registered before any auth middleware so it is accessible without a session.
func (h *Handler) Register(app *fiber.App) {
	app.Get(Path, h.Check)
}

type status struct {
	Status   string            `json:"status"`
	Checks   map[string]string `json:"checks"`
}

// Check responds with 200 OK when the service is healthy, or 503 Service
// Unavailable during graceful shutdown or when a dependency is unhealthy.
func (h *Handler) Check(c fiber.Ctx) error {
	checks := make(map[string]string)
	healthy := true

	// Liveness: set to false during graceful shutdown.
	if !h.alive.Load() {
		checks["alive"] = "shutting_down"
		healthy = false
	} else {
		checks["alive"] = "ok"
	}

	// Database connectivity.
	if err := h.probeDB(); err != nil {
		checks["database"] = "error"
		healthy = false
	} else {
		checks["database"] = "ok"
	}

	// PowerDNS client (optional — absence is not fatal, just reported).
	if powerdns.Engine.Client == nil {
		checks["powerdns"] = "not_configured"
	} else {
		checks["powerdns"] = "ok"
	}

	s := status{Checks: checks}

	if healthy {
		s.Status = "ok"
		return c.Status(fiber.StatusOK).JSON(s)
	}

	s.Status = "degraded"

	return c.Status(fiber.StatusServiceUnavailable).JSON(s)
}

func (h *Handler) probeDB() error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetConnMaxLifetime(dbProbeTimeout)

	return sqlDB.Ping()
}
