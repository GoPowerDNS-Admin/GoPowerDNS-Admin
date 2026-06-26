// Package branding implements the admin GUI for editing branding settings
// (product name, logo and favicons) and serves the uploaded assets.
package branding

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/auth"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	controller "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/branding"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/dashboard"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/navigation"
)

const (
	// Path is the path to the branding settings page.
	Path = handler.BrandingSettingsPath

	// TemplateName is the name of the branding settings template.
	TemplateName = "admin/settings/branding"

	// maxUploadBytes is the maximum accepted size for an uploaded image.
	maxUploadBytes = 1 << 20 // 1 MiB

	// svgSniffLen is how many leading bytes are inspected when detecting SVG.
	svgSniffLen = 1024
)

// errNoFile signals that no file was submitted for an optional upload field.
var errNoFile = errors.New("no file uploaded")

// Service is the branding settings handler service.
type Service struct {
	handler.Service
	cfg   *config.Config
	db    *gorm.DB
	store *controller.Store
}

// Handler is the branding settings handler.
var Handler = Service{}

// Init initializes the branding settings handler. The shared store is created
// in the web service and also drives the template branding injection.
func (s *Service) Init(
	app *fiber.App,
	cfg *config.Config,
	db *gorm.DB,
	authService *auth.Service,
	store *controller.Store,
) {
	if app == nil || cfg == nil || db == nil || store == nil {
		log.Fatal().Msg(handler.ErrNilACDFatalLogMsg)
		return
	}

	s.db = db
	s.cfg = cfg
	s.store = store

	// settings page (permission protected)
	app.Get(Path, auth.RequirePermission(authService, auth.PermAdminBranding), s.Get)
	app.Post(Path, auth.RequirePermission(authService, auth.PermAdminBranding), s.Post)

	// asset-serving routes — public (favicons/logo load on the unauthenticated
	// login and TOTP pages); the auth middleware allowlists the /branding prefix.
	app.Get(controller.LogoPath, s.serveAsset(controller.SlotLogo))
	app.Get(controller.FaviconSVGPath, s.serveAsset(controller.SlotFaviconSVG))
	app.Get(controller.FaviconPNGPath, s.serveAsset(controller.SlotFaviconPNG))
}

// Get renders the branding settings form.
func (s *Service) Get(c fiber.Ctx) error {
	return c.Render(TemplateName, s.viewData(nil, nil), handler.BaseLayout)
}

// Post handles the branding settings form submission (multipart).
func (s *Service) Post(c fiber.Ctx) error {
	// Start from the current settings so existing uploads are preserved when a
	// field is left untouched.
	cur := *s.store.Settings()

	cur.Name = strings.TrimSpace(c.FormValue("name"))
	cur.LogoURL = strings.TrimSpace(c.FormValue("logo_url"))
	cur.FaviconURL = strings.TrimSpace(c.FormValue("favicon_url"))
	cur.FaviconPNGURL = strings.TrimSpace(c.FormValue("favicon_png_url"))

	type slot struct {
		field   string
		remove  string
		dst     **controller.Asset
		require imageKind
	}

	slots := []slot{
		{"logo", "remove_logo", &cur.Logo, kindImage},
		{"favicon_svg", "remove_favicon_svg", &cur.FaviconSVG, kindSVG},
		{"favicon_png", "remove_favicon_png", &cur.FaviconPNG, kindPNG},
	}

	for _, sl := range slots {
		if c.FormValue(sl.remove) != "" {
			*sl.dst = nil
		}

		asset, err := s.readUpload(c, sl.field, sl.require)

		switch {
		case errors.Is(err, errNoFile):
			// nothing uploaded for this field; keep the existing value
		case err != nil:
			return s.renderError(c, &cur, err.Error())
		default:
			*sl.dst = asset
		}
	}

	if err := cur.Save(s.db); err != nil {
		log.Error().Err(err).Msg("failed to save branding settings")

		return s.renderError(c, &cur, "Failed to save settings")
	}

	if err := s.store.Reload(); err != nil {
		log.Error().Err(err).Msg("failed to reload branding settings after save")
	}

	log.Info().Msg("branding settings saved successfully")

	return c.Render(TemplateName, s.viewData(nil, fiber.Map{"Success": "Branding saved successfully"}), handler.BaseLayout)
}

// serveAsset returns a handler that serves the uploaded image for the slot.
func (s *Service) serveAsset(slotName string) fiber.Handler {
	return func(c fiber.Ctx) error {
		asset := s.store.Asset(slotName)
		if asset == nil || len(asset.Data) == 0 {
			return c.SendStatus(fiber.StatusNotFound)
		}

		if match := c.Get("If-None-Match"); match != "" && match == asset.ETag {
			return c.SendStatus(fiber.StatusNotModified)
		}

		c.Set(fiber.HeaderContentType, asset.ContentType)
		c.Set(fiber.HeaderCacheControl, "public, max-age=300")
		c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
		c.Set(fiber.HeaderETag, asset.ETag)

		return c.Send(asset.Data)
	}
}

// readUpload reads and validates an optional uploaded file. It returns errNoFile
// when no file was submitted for the field.
func (s *Service) readUpload(c fiber.Ctx, field string, require imageKind) (*controller.Asset, error) {
	fh, err := c.FormFile(field)
	if err != nil || fh == nil {
		return nil, errNoFile
	}

	if fh.Size > maxUploadBytes {
		return nil, &uploadError{"Image too large (max 1 MB): " + fh.Filename}
	}

	data, err := readFileHeader(fh)
	if err != nil {
		return nil, &uploadError{"Failed to read uploaded file: " + fh.Filename}
	}

	contentType, err := detectImage(data, require)
	if err != nil {
		return nil, err
	}

	if err := validateSquareFavicon(data, require); err != nil {
		return nil, err
	}

	return controller.NewAsset(contentType, data), nil
}

func readFileHeader(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	return io.ReadAll(io.LimitReader(f, maxUploadBytes))
}

// viewData builds the template payload, merging any extra keys (e.g. Success/Error).
func (s *Service) viewData(current *controller.Settings, extra fiber.Map) fiber.Map {
	if current == nil {
		current = s.store.Settings()
	}

	nav := navigation.NewContext("Branding", "settings", "branding").
		AddBreadcrumb("Home", dashboard.Path, false).
		AddBreadcrumb("Settings", "", false).
		AddBreadcrumb("Branding", Path, true)

	data := fiber.Map{
		"Navigation": nav,
		"Current":    current,
		"Brand":      s.store.Brand(),
	}

	for k, v := range extra {
		data[k] = v
	}

	return data
}

// renderError re-renders the form with the unsaved values and an error message.
func (s *Service) renderError(c fiber.Ctx, current *controller.Settings, msg string) error {
	return c.Status(fiber.StatusBadRequest).
		Render(TemplateName, s.viewData(current, fiber.Map{"Error": msg}), handler.BaseLayout)
}

// imageKind constrains which image types are accepted for a given slot.
type imageKind int

const (
	kindImage imageKind = iota // any common web image (incl. SVG)
	kindSVG                    // SVG only
	kindPNG                    // PNG only
)

type uploadError struct{ msg string }

func (e *uploadError) Error() string { return e.msg }

// detectImage sniffs the uploaded bytes and returns the canonical content type,
// enforcing the required kind. SVG is detected separately because
// http.DetectContentType reports XML/plain text for it.
func detectImage(data []byte, require imageKind) (string, error) {
	if len(data) == 0 {
		return "", &uploadError{"Uploaded file is empty"}
	}

	svg := looksLikeSVG(data)
	sniffed := http.DetectContentType(data)

	switch require {
	case kindSVG:
		if !svg {
			return "", &uploadError{"Favicon (SVG) must be an SVG image"}
		}

		return "image/svg+xml", nil
	case kindPNG:
		if sniffed != "image/png" {
			return "", &uploadError{"Favicon (PNG) must be a PNG image"}
		}

		return "image/png", nil
	case kindImage:
		if svg {
			return "image/svg+xml", nil
		}

		if strings.HasPrefix(sniffed, "image/") {
			return sniffed, nil
		}

		return "", &uploadError{"Logo must be an image (SVG, PNG, JPEG, GIF, WebP or ICO)"}
	default:
		return "", &uploadError{"Unsupported image type"}
	}
}

// looksLikeSVG reports whether data appears to be an SVG document.
func looksLikeSVG(data []byte) bool {
	head := data
	if len(head) > svgSniffLen {
		head = head[:svgSniffLen]
	}

	return bytes.Contains(bytes.ToLower(head), []byte("<svg"))
}

var (
	svgViewBoxRe = regexp.MustCompile(`(?i)viewbox\s*=\s*["']\s*[-\d.eE]+\s+[-\d.eE]+\s+([-\d.eE]+)\s+([-\d.eE]+)`)
	svgWidthRe   = regexp.MustCompile(`(?i)\swidth\s*=\s*["']\s*([\d.eE]+)`)
	svgHeightRe  = regexp.MustCompile(`(?i)\sheight\s*=\s*["']\s*([\d.eE]+)`)
)

// validateSquareFavicon enforces a square aspect ratio for favicon uploads.
// PNG dimensions are read from the header; SVG dimensions are derived from the
// viewBox (preferred) or width/height attributes. SVGs whose dimensions cannot
// be determined are accepted, since they scale freely. The logo is unconstrained.
func validateSquareFavicon(data []byte, require imageKind) error {
	switch require {
	case kindPNG:
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return &uploadError{"Could not read PNG dimensions"}
		}

		if cfg.Width != cfg.Height {
			return &uploadError{fmt.Sprintf("Favicon (PNG) must be square (got %dx%d)", cfg.Width, cfg.Height)}
		}
	case kindSVG:
		if w, h, ok := svgDimensions(data); ok && w != h {
			return &uploadError{fmt.Sprintf("Favicon (SVG) must be square (got %sx%s)",
				strconv.FormatFloat(w, 'f', -1, 64), strconv.FormatFloat(h, 'f', -1, 64))}
		}
	case kindImage:
		// logo: no aspect-ratio constraint
	}

	return nil
}

// svgDimensions extracts the width/height of an SVG from its viewBox, falling
// back to the width/height attributes. ok is false when neither is present.
func svgDimensions(data []byte) (width, height float64, ok bool) {
	head := data
	if len(head) > svgSniffLen {
		head = head[:svgSniffLen]
	}

	if m := svgViewBoxRe.FindSubmatch(head); m != nil {
		w, errW := strconv.ParseFloat(string(m[1]), 64)
		h, errH := strconv.ParseFloat(string(m[2]), 64)

		if errW == nil && errH == nil {
			return w, h, true
		}
	}

	mw := svgWidthRe.FindSubmatch(head)
	mh := svgHeightRe.FindSubmatch(head)

	if mw != nil && mh != nil {
		w, errW := strconv.ParseFloat(string(mw[1]), 64)
		h, errH := strconv.ParseFloat(string(mh[1]), 64)

		if errW == nil && errH == nil {
			return w, h, true
		}
	}

	return 0, 0, false
}
