// Package branding provides runtime-editable branding settings (product name,
// logo and favicons) stored in the database, overriding the static TOML
// configuration. A Store caches the resolved branding in memory so it can be
// injected into every rendered template without a per-request database query.
package branding

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync/atomic"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
)

const (
	// SettingKey is the key under which branding settings are stored.
	SettingKey = "branding"

	// LogoPath is the route that serves an uploaded logo.
	LogoPath = "/branding/logo"
	// FaviconSVGPath is the route that serves an uploaded SVG favicon.
	FaviconSVGPath = "/branding/favicon.svg"
	// FaviconPNGPath is the route that serves an uploaded PNG favicon.
	FaviconPNGPath = "/branding/favicon.png"

	// SlotLogo identifies the logo asset.
	SlotLogo = "logo"
	// SlotFaviconSVG identifies the SVG favicon asset.
	SlotFaviconSVG = "favicon-svg"
	// SlotFaviconPNG identifies the PNG favicon asset.
	SlotFaviconPNG = "favicon-png"
)

// Asset is an uploaded image stored in the database.
type Asset struct {
	ContentType string `json:"content_type"`
	ETag        string `json:"etag"`
	Data        []byte `json:"data"`
}

// NewAsset builds an Asset and derives a short content hash used as an ETag and
// cache-busting query value.
func NewAsset(contentType string, data []byte) *Asset {
	sum := sha256.Sum256(data)

	return &Asset{
		ContentType: contentType,
		ETag:        hex.EncodeToString(sum[:8]),
		Data:        data,
	}
}

// Settings holds the persisted branding overrides. Any empty URL field and any
// nil asset falls back to the TOML config and finally the bundled defaults when
// resolved. An uploaded asset takes precedence over the matching URL field.
type Settings struct {
	Name          string `json:"name"`
	LogoURL       string `json:"logo_url"`
	FaviconURL    string `json:"favicon_url"`
	FaviconPNGURL string `json:"favicon_png_url"`

	Logo       *Asset `json:"logo,omitempty"`
	FaviconSVG *Asset `json:"favicon_svg,omitempty"`
	FaviconPNG *Asset `json:"favicon_png,omitempty"`
}

// Load reads the branding settings from the database. It returns
// setting.ErrSettingNotFound when nothing has been saved yet.
func Load(db *gorm.DB) (*Settings, error) {
	s, err := setting.Get(db, SettingKey)
	if err != nil {
		return nil, err
	}

	var out Settings
	if err := json.Unmarshal(s.Value, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Save persists the branding settings to the database (upsert).
func (s *Settings) Save(db *gorm.DB) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	_, err = setting.Set(db, SettingKey, data)

	return err
}

// Store caches branding settings in memory and resolves them against the static
// TOML configuration. It is safe for concurrent use.
type Store struct {
	db      *gorm.DB
	base    config.Branding
	title   string
	current atomic.Pointer[Settings]
}

// NewStore creates a Store and loads the current settings from the database.
// base and title are the TOML-configured branding and application title used as
// fallbacks. A non-nil error indicates the initial load failed; the returned
// Store is still usable and falls back to the TOML/default branding.
func NewStore(db *gorm.DB, base config.Branding, title string) (*Store, error) {
	st := &Store{db: db, base: base, title: title}
	st.current.Store(&Settings{})

	err := st.Reload()

	return st, err
}

// Reload refreshes the cached settings from the database. A missing setting is
// not an error: the cache is reset to empty so resolution falls back to the
// TOML/default branding.
func (st *Store) Reload() error {
	s, err := Load(st.db)
	if err != nil {
		if errors.Is(err, setting.ErrSettingNotFound) {
			st.current.Store(&Settings{})

			return nil
		}

		return err
	}

	st.current.Store(s)

	return nil
}

// Settings returns the cached raw settings. The returned pointer must be
// treated as read-only.
func (st *Store) Settings() *Settings {
	if s := st.current.Load(); s != nil {
		return s
	}

	return &Settings{}
}

// Asset returns the uploaded asset for the given slot, or nil if none is set.
func (st *Store) Asset(slot string) *Asset {
	s := st.Settings()

	switch slot {
	case SlotLogo:
		return s.Logo
	case SlotFaviconSVG:
		return s.FaviconSVG
	case SlotFaviconPNG:
		return s.FaviconPNG
	default:
		return nil
	}
}

// Brand resolves the branding for template rendering. Precedence per field is:
// uploaded asset → DB URL → TOML config → bundled default.
func (st *Store) Brand() config.Branding {
	out := st.base.Resolve(st.title)
	s := st.Settings()

	if s.Name != "" {
		out.Name = s.Name
	}

	out.LogoURL = pick(s.Logo, LogoPath, s.LogoURL, out.LogoURL)
	out.FaviconURL = pick(s.FaviconSVG, FaviconSVGPath, s.FaviconURL, out.FaviconURL)
	out.FaviconPNGURL = pick(s.FaviconPNG, FaviconPNGPath, s.FaviconPNGURL, out.FaviconPNGURL)

	return out
}

// pick chooses the effective URL for an image: the asset-serving route (with a
// cache-busting hash) when an asset is uploaded, otherwise the DB URL, otherwise
// the already-resolved fallback.
func pick(a *Asset, path, dbURL, fallback string) string {
	if a != nil && len(a.Data) > 0 {
		return path + "?v=" + a.ETag
	}

	if dbURL != "" {
		return dbURL
	}

	return fallback
}
