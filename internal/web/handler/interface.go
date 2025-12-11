package handler

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
)

// Service is the interface for a web handler service.
type Service interface {
	Init(app *fiber.App, cfg *config.Config, db *gorm.DB) error
}
