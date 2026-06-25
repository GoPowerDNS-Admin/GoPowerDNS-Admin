package branding

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Setting{}))

	return db
}

func TestStore_Brand_FallsBackToConfigAndDefaults(t *testing.T) {
	db := setupTestDB(t)

	// No DB setting yet: resolve against TOML config and bundled defaults.
	st, err := NewStore(db, config.Branding{Name: "Acme DNS"}, "Title Fallback")
	require.NoError(t, err)

	b := st.Brand()
	require.Equal(t, "Acme DNS", b.Name)
	require.Equal(t, config.DefaultLogoURL, b.LogoURL)
	require.Equal(t, config.DefaultLogoURL, b.FaviconURL)
	require.Equal(t, config.DefaultFaviconPNGURL, b.FaviconPNGURL)
}

func TestStore_Brand_DBOverridesConfig(t *testing.T) {
	db := setupTestDB(t)

	// Seed a DB override with a URL and an uploaded logo asset.
	seed := &Settings{
		Name:    "DB Brand",
		LogoURL: "/static/img/db-logo.svg",
		Logo:    NewAsset("image/png", []byte("\x89PNGfakebytes")),
	}
	require.NoError(t, seed.Save(db))

	st, err := NewStore(db, config.Branding{Name: "Config Brand"}, "Title")
	require.NoError(t, err)

	b := st.Brand()
	require.Equal(t, "DB Brand", b.Name, "DB name should override config")

	// Uploaded asset wins over the URL field and points at the serving route
	// with a cache-busting hash.
	require.True(t, strings.HasPrefix(b.LogoURL, LogoPath+"?v="), "uploaded logo should be served from %s, got %s", LogoPath, b.LogoURL)

	// Favicons had no override → fall back to defaults.
	require.Equal(t, config.DefaultLogoURL, b.FaviconURL)
	require.Equal(t, config.DefaultFaviconPNGURL, b.FaviconPNGURL)
}

func TestStore_Brand_DBURLWithoutAsset(t *testing.T) {
	db := setupTestDB(t)

	seed := &Settings{FaviconPNGURL: "/static/img/custom.png"}
	require.NoError(t, seed.Save(db))

	st, err := NewStore(db, config.Branding{}, "Title")
	require.NoError(t, err)

	require.Equal(t, "/static/img/custom.png", st.Brand().FaviconPNGURL)
}

func TestStore_Asset_RoundTripAndReload(t *testing.T) {
	db := setupTestDB(t)

	st, err := NewStore(db, config.Branding{}, "Title")
	require.NoError(t, err)
	require.Nil(t, st.Asset(SlotLogo), "no asset before any save")

	seed := &Settings{FaviconSVG: NewAsset("image/svg+xml", []byte("<svg></svg>"))}
	require.NoError(t, seed.Save(db))

	// Cache is stale until reloaded.
	require.Nil(t, st.Asset(SlotFaviconSVG))
	require.NoError(t, st.Reload())

	got := st.Asset(SlotFaviconSVG)
	require.NotNil(t, got)
	require.Equal(t, "image/svg+xml", got.ContentType)
	require.Equal(t, []byte("<svg></svg>"), got.Data)
	require.NotEmpty(t, got.ETag)
}
