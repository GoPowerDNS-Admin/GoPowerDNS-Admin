// Package navigation provides utilities for managing navigation state and breadcrumbs.
package navigation

// BreadcrumbItem represents a single breadcrumb link.
type BreadcrumbItem struct {
	Title  string
	URL    string
	Active bool
}

// Context represents the navigation context for a page.
type Context struct {
	ActiveSection string
	ActivePage    string
	Breadcrumbs   []BreadcrumbItem
	PageTitle     string
}

// NewContext creates a new navigation context.
func NewContext(pageTitle, activeSection, activePage string) *Context {
	return &Context{
		PageTitle:     pageTitle,
		ActiveSection: activeSection,
		ActivePage:    activePage,
		Breadcrumbs:   make([]BreadcrumbItem, 0),
	}
}

// AddBreadcrumb adds a breadcrumb item to the context.
func (c *Context) AddBreadcrumb(title, url string, active bool) *Context {
	c.Breadcrumbs = append(c.Breadcrumbs, BreadcrumbItem{
		Title:  title,
		URL:    url,
		Active: active,
	})

	return c
}

// IsActive checks if the given section and page match the current context.
func (c *Context) IsActive(section, page string) bool {
	return c.ActiveSection == section && c.ActivePage == page
}

// IsSectionActive checks if the given section is active.
func (c *Context) IsSectionActive(section string) bool {
	return c.ActiveSection == section
}
