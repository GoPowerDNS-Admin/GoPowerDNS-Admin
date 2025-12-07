package navigation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext("Test Page", "section1", "page1")

	assert.Equal(t, "Test Page", ctx.PageTitle)
	assert.Equal(t, "section1", ctx.ActiveSection)
	assert.Equal(t, "page1", ctx.ActivePage)
	assert.NotNil(t, ctx.Breadcrumbs)
	assert.Empty(t, ctx.Breadcrumbs)
}

func TestContext_AddBreadcrumb(t *testing.T) {
	ctx := NewContext("Test Page", "section1", "page1")

	// Add first breadcrumb
	ctx.AddBreadcrumb("Home", "/", false)
	assert.Len(t, ctx.Breadcrumbs, 1)
	assert.Equal(t, "Home", ctx.Breadcrumbs[0].Title)
	assert.Equal(t, "/", ctx.Breadcrumbs[0].URL)
	assert.False(t, ctx.Breadcrumbs[0].Active)

	// Add second breadcrumb
	ctx.AddBreadcrumb("Settings", "/settings", false)
	assert.Len(t, ctx.Breadcrumbs, 2)
	assert.Equal(t, "Settings", ctx.Breadcrumbs[1].Title)

	// Add active breadcrumb
	ctx.AddBreadcrumb("Current Page", "/settings/page", true)
	assert.Len(t, ctx.Breadcrumbs, 3)
	assert.True(t, ctx.Breadcrumbs[2].Active)
}

func TestContext_AddBreadcrumb_Chaining(t *testing.T) {
	ctx := NewContext("Test Page", "section1", "page1").
		AddBreadcrumb("Home", "/", false).
		AddBreadcrumb("Settings", "/settings", false).
		AddBreadcrumb("Current", "/settings/current", true)

	assert.Len(t, ctx.Breadcrumbs, 3)
	assert.Equal(t, "Home", ctx.Breadcrumbs[0].Title)
	assert.Equal(t, "Settings", ctx.Breadcrumbs[1].Title)
	assert.Equal(t, "Current", ctx.Breadcrumbs[2].Title)
	assert.True(t, ctx.Breadcrumbs[2].Active)
}

func TestContext_IsActive(t *testing.T) {
	ctx := NewContext("Test Page", "settings", "pdns-server")

	// Should return true when both section and page match
	assert.True(t, ctx.IsActive("settings", "pdns-server"))

	// Should return false when section doesn't match
	assert.False(t, ctx.IsActive("dashboard", "pdns-server"))

	// Should return false when page doesn't match
	assert.False(t, ctx.IsActive("settings", "basic"))

	// Should return false when neither match
	assert.False(t, ctx.IsActive("dashboard", "main"))
}

func TestContext_IsSectionActive(t *testing.T) {
	ctx := NewContext("Test Page", "settings", "pdns-server")

	// Should return true when section matches
	assert.True(t, ctx.IsSectionActive("settings"))

	// Should return false when section doesn't match
	assert.False(t, ctx.IsSectionActive("dashboard"))
	assert.False(t, ctx.IsSectionActive("admin"))
}

func TestBreadcrumbItem(t *testing.T) {
	item := BreadcrumbItem{
		Title:  "Test",
		URL:    "/test",
		Active: true,
	}

	assert.Equal(t, "Test", item.Title)
	assert.Equal(t, "/test", item.URL)
	assert.True(t, item.Active)
}
