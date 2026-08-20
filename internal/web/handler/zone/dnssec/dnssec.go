// Package dnssec provides HTTP handlers for DNSSEC lifecycle management.
// Phase 1 covers: key visibility, enable/disable DNSSEC, toggle active/published
// state per key, delete individual keys, and DS record display.
package dnssec

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/activitylog"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/session"
)

const (
	// BasePath is the route prefix for all DNSSEC endpoints.
	// :name is the zone name (with or without trailing dot — normalised server-side).
	BasePath = handler.RootPath + "zone/edit/:name/dnssec"

	defaultTimeout = 30 * time.Second
)

// Service is the DNSSEC handler service.
type Service struct {
	handler.Service
	cfg         *config.Config
	db          *gorm.DB
	authService *auth.Service
}

// Handler is the singleton DNSSEC handler.
var Handler = Service{}

// Init registers all DNSSEC routes with the Fiber app.
// All routes require PermZoneUpdate — the same permission used by the zone edit handler.
func (s *Service) Init(app *fiber.App, cfg *config.Config, db *gorm.DB, authService *auth.Service) {
	if app == nil || cfg == nil || db == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.cfg = cfg
	s.db = db
	s.authService = authService

	requireUpdate := auth.RequirePermission(authService, auth.PermZoneUpdate)

	// GET  /zone/edit/:name/dnssec        — list cryptokeys (JSON)
	app.Get(BasePath, requireUpdate, s.List)

	// POST /zone/edit/:name/dnssec/enable  — enable DNSSEC (create first CSK)
	app.Post(BasePath+"/enable", requireUpdate, s.Enable)

	// POST /zone/edit/:name/dnssec/disable — disable DNSSEC (delete all keys)
	app.Post(BasePath+"/disable", requireUpdate, s.Disable)

	// POST /zone/edit/:name/dnssec/keys/:id/toggle — toggle active/published
	app.Post(BasePath+"/keys/:id/toggle", requireUpdate, s.ToggleKey)

	// POST /zone/edit/:name/dnssec/keys/:id/delete — delete a single key
	app.Post(BasePath+"/keys/:id/delete", requireUpdate, s.DeleteKey)

	// GET /zone/edit/:name/dnssec/chain — full trust chain verification
	app.Get(BasePath+"/chain", requireUpdate, s.CheckChain)
}

// CryptokeyView is the JSON representation of a cryptokey sent to the frontend.
type CryptokeyView struct {
	ID        uint64   `json:"id"`
	KeyType   string   `json:"key_type"`
	Algorithm string   `json:"algorithm"`
	Bits      uint64   `json:"bits"`
	Active    bool     `json:"active"`
	Published bool     `json:"published"`
	DNSKey    string   `json:"dnskey"`
	DS        []string `json:"ds"`
}

// List returns the cryptokeys for a zone as JSON.
// Handles GET /zone/edit/:name/dnssec.
func (s *Service) List(c fiber.Ctx) error {
	zoneName, err := s.zoneNameFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if !s.canAccessZone(c, zoneName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "access denied"})
	}

	if powerdns.Engine.Client == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": powerdns.ErrMsgClientNotInitialized,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	keys, err := powerdns.Engine.Cryptokeys.List(ctx, zoneName)
	if err != nil {
		log.Error().Err(err).Str("zone", zoneName).Msg("failed to list cryptokeys")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": fmt.Sprintf("failed to list cryptokeys: %v", err),
		})
	}

	views := make([]CryptokeyView, 0, len(keys))
	for _, k := range keys {
		views = append(views, cryptokeyToView(&k))
	}

	return c.JSON(fiber.Map{"success": true, "keys": views})
}

// Enable activates DNSSEC on a zone by creating a Combined Signing Key (CSK).
// Handles POST /zone/edit/:name/dnssec/enable.
func (s *Service) Enable(c fiber.Ctx) error {
	zoneName, err := s.zoneNameFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if !s.canAccessZone(c, zoneName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "access denied"})
	}

	if powerdns.Engine.Client == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": powerdns.ErrMsgClientNotInitialized,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	keyType := "csk"
	active := true
	published := true

	key, err := powerdns.Engine.Cryptokeys.Create(ctx, zoneName, pdnsapi.Cryptokey{
		KeyType:   &keyType,
		Active:    &active,
		Published: &published,
	})
	if err != nil {
		log.Error().Err(err).Str("zone", zoneName).Msg("failed to enable DNSSEC")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": fmt.Sprintf("failed to enable DNSSEC: %v", err),
		})
	}

	log.Info().Str("zone", zoneName).Msg("DNSSEC enabled")

	userID, username := currentUserFromSession(c)
	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       userID,
		Username:     username,
		Action:       activitylog.ActionDNSSECEnabled,
		ResourceType: activitylog.ResourceTypeZone,
		ResourceName: zoneName,
		Details:      cryptokeyToView(key),
		IPAddress:    c.IP(),
	})

	return c.JSON(fiber.Map{"success": true, "key": cryptokeyToView(key)})
}

// Disable removes all cryptokeys from a zone, effectively disabling DNSSEC.
// Handles POST /zone/edit/:name/dnssec/disable.
func (s *Service) Disable(c fiber.Ctx) error {
	zoneName, err := s.zoneNameFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if !s.canAccessZone(c, zoneName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "access denied"})
	}

	if powerdns.Engine.Client == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": powerdns.ErrMsgClientNotInitialized,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	keys, err := powerdns.Engine.Cryptokeys.List(ctx, zoneName)
	if err != nil {
		log.Error().Err(err).Str("zone", zoneName).Msg("failed to list cryptokeys for disable")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": fmt.Sprintf("failed to list cryptokeys: %v", err),
		})
	}

	for _, k := range keys {
		if k.ID == nil {
			continue
		}

		if delErr := powerdns.Engine.Cryptokeys.Delete(ctx, zoneName, *k.ID); delErr != nil {
			log.Error().Err(delErr).Str("zone", zoneName).Uint64("key_id", *k.ID).Msg("failed to delete cryptokey")

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("failed to delete key %d: %v", *k.ID, delErr),
			})
		}
	}

	log.Info().Str("zone", zoneName).Int("keys_removed", len(keys)).Msg("DNSSEC disabled")

	userID, username := currentUserFromSession(c)
	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       userID,
		Username:     username,
		Action:       activitylog.ActionDNSSECDisabled,
		ResourceType: activitylog.ResourceTypeZone,
		ResourceName: zoneName,
		Details:      fiber.Map{"keys_removed": len(keys)},
		IPAddress:    c.IP(),
	})

	return c.JSON(fiber.Map{"success": true, "message": "DNSSEC disabled"})
}

// ToggleKeyRequest is the JSON body for the toggle endpoint.
type ToggleKeyRequest struct {
	Active    *bool `json:"active"`
	Published *bool `json:"published"`
}

// ToggleKey changes the active and/or published state of a single cryptokey.
// Handles POST /zone/edit/:name/dnssec/keys/:id/toggle.
func (s *Service) ToggleKey(c fiber.Ctx) error {
	zoneName, err := s.zoneNameFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	keyID, err := keyIDFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if !s.canAccessZone(c, zoneName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "access denied"})
	}

	var req ToggleKeyRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid request body"})
	}

	if req.Active == nil && req.Published == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "message": "at least one of 'active' or 'published' must be provided",
		})
	}

	if powerdns.Engine.Client == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": powerdns.ErrMsgClientNotInitialized,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	update := pdnsapi.Cryptokey{
		Active:    req.Active,
		Published: req.Published,
	}

	if err := powerdns.Engine.Cryptokeys.Change(ctx, zoneName, keyID, update); err != nil {
		log.Error().Err(err).Str("zone", zoneName).Uint64("key_id", keyID).Msg("failed to toggle cryptokey")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": fmt.Sprintf("failed to update key: %v", err),
		})
	}

	log.Info().Str("zone", zoneName).Uint64("key_id", keyID).Msg("cryptokey toggled")

	userID, username := currentUserFromSession(c)
	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       userID,
		Username:     username,
		Action:       activitylog.ActionDNSSECKeyChanged,
		ResourceType: activitylog.ResourceTypeZone,
		ResourceName: zoneName,
		Details:      fiber.Map{"key_id": keyID, "active": req.Active, "published": req.Published},
		IPAddress:    c.IP(),
	})

	return c.JSON(fiber.Map{"success": true, "message": "key updated"})
}

// DeleteKey removes a single cryptokey by ID.
// Handles POST /zone/edit/:name/dnssec/keys/:id/delete.
func (s *Service) DeleteKey(c fiber.Ctx) error {
	zoneName, err := s.zoneNameFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	keyID, err := keyIDFromParam(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if !s.canAccessZone(c, zoneName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "access denied"})
	}

	if powerdns.Engine.Client == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": powerdns.ErrMsgClientNotInitialized,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if err := powerdns.Engine.Cryptokeys.Delete(ctx, zoneName, keyID); err != nil {
		log.Error().Err(err).Str("zone", zoneName).Uint64("key_id", keyID).Msg("failed to delete cryptokey")

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "message": fmt.Sprintf("failed to delete key: %v", err),
		})
	}

	log.Info().Str("zone", zoneName).Uint64("key_id", keyID).Msg("cryptokey deleted")

	userID, username := currentUserFromSession(c)
	activitylog.Record(&activitylog.Entry{
		DB:           s.db,
		UserID:       userID,
		Username:     username,
		Action:       activitylog.ActionDNSSECKeyDeleted,
		ResourceType: activitylog.ResourceTypeZone,
		ResourceName: zoneName,
		Details:      fiber.Map{"key_id": keyID},
		IPAddress:    c.IP(),
	})

	return c.JSON(fiber.Map{"success": true, "message": "key deleted"})
}

// zoneNameFromParam extracts and normalises the zone name from the :name route param.
func (s *Service) zoneNameFromParam(c fiber.Ctx) (string, error) {
	name := c.Params("name")
	if name == "" {
		return "", errors.New("zone name is required")
	}

	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	return name, nil
}

// keyIDFromParam parses the :id route param as a uint64.
func keyIDFromParam(c fiber.Ctx) (uint64, error) {
	idStr := c.Params("id")

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid key id: %q", idStr)
	}

	return id, nil
}

// canAccessZone returns false when zone-tag RBAC restrictions prevent access.
func (s *Service) canAccessZone(c fiber.Ctx, zoneName string) bool {
	user, ok := c.Locals("CurrentUser").(models.User)
	if !ok || user.ID == 0 {
		return false
	}

	if s.authService == nil {
		return true
	}

	accessible, err := s.authService.GetAccessibleZoneIDs(user.ID)
	if err != nil || accessible == nil {
		return true
	}

	return accessible[zoneName]
}

// currentUserFromSession extracts user ID and username from the session cookie.
func currentUserFromSession(c fiber.Ctx) (*uint64, string) {
	sid := c.Cookies("session")
	if sid == "" {
		return nil, ""
	}

	sd := new(session.Data)
	if err := sd.Read(sid); err != nil || sd.User.ID == 0 {
		return nil, ""
	}

	id := sd.User.ID

	return &id, sd.User.Username
}

// cryptokeyToView converts a pdnsapi.Cryptokey to a CryptokeyView for the frontend.
func cryptokeyToView(k *pdnsapi.Cryptokey) CryptokeyView {
	v := CryptokeyView{}

	if k.ID != nil {
		v.ID = *k.ID
	}

	if k.KeyType != nil {
		v.KeyType = *k.KeyType
	}

	if k.Algorithm != nil {
		v.Algorithm = *k.Algorithm
	}

	if k.Bits != nil {
		v.Bits = *k.Bits
	}

	if k.Active != nil {
		v.Active = *k.Active
	}

	if k.Published != nil {
		v.Published = *k.Published
	}

	if k.DNSkey != nil {
		v.DNSKey = *k.DNSkey
	}

	v.DS = k.DS

	return v
}
