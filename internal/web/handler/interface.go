package handler

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

const (
	// RootPath is the root path for all handlers.
	RootPath = "/"

	// RouterRootPath is the root path the route group.
	RouterRootPath = "/"
)

// Service is the interface for a web handler service.
type Service interface {
	Init(app *fiber.App, cfg *config.Config, db *gorm.DB) error
}
