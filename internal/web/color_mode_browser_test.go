//go:build browser

package web_test

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// colorModeTestPage mirrors the colour mode switch in
// partials/header-navigation.gohtml and the script order of the real layout.
const colorModeTestPage = `<!DOCTYPE html>
<html lang="en">
<head>
<script src="/static/js/theme-init.js"></script>
<meta charset="UTF-8">
</head>
<body>
<ul>
  <li class="nav-item dropdown">
    <a class="nav-link" href="#" id="color-mode-toggle" data-bs-toggle="dropdown" aria-expanded="false">
      <i class="bi bi-sun-fill" data-lte-theme-icon="light"></i>
      <i class="bi bi-moon-fill d-none" data-lte-theme-icon="dark"></i>
      <i class="bi bi-circle-half d-none" data-lte-theme-icon="auto"></i>
    </a>
    <ul class="dropdown-menu">
      <li><button type="button" class="dropdown-item" data-bs-theme-value="light">Light</button></li>
      <li><button type="button" class="dropdown-item" data-bs-theme-value="dark">Dark</button></li>
      <li><button type="button" class="dropdown-item" data-bs-theme-value="auto">Auto</button></li>
    </ul>
  </li>
</ul>
<script src="/static/vendor/bootstrap-5.3.9-dist/js/bootstrap.bundle.min.js"></script>
<script src="/static/vendor/adminlte-v4/js/adminlte.min.js"></script>
</body>
</html>`

// themeInitOnlyPage loads only theme-init.js, to show it applies the saved
// mode on its own, before AdminLTE's script runs.
const themeInitOnlyPage = `<!DOCTYPE html>
<html lang="en">
<head><script src="/static/js/theme-init.js"></script></head>
<body></body>
</html>`

func newColorModeServer(t *testing.T) *httptest.Server {
	t.Helper()

	staticFS, err := fs.Sub(os.DirFS("static"), ".")
	if err != nil {
		t.Fatalf("open static dir: %v", err)
	}

	page := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(body))
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/init-only", page(themeInitOnlyPage))
	mux.HandleFunc("/", page(colorModeTestPage))

	return httptest.NewServer(mux)
}

func newColorModeContext(t *testing.T) context.Context {
	t.Helper()

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts()...)
	t.Cleanup(allocCancel)

	ctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(cancel)

	return ctx
}

func systemColorScheme(scheme string) chromedp.Action {
	return emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{
		{Name: "prefers-color-scheme", Value: scheme},
	})
}

func pickColorMode(mode string) chromedp.Action {
	item := `[data-bs-theme-value="` + mode + `"]`

	return chromedp.Tasks{
		chromedp.Click(`#color-mode-toggle`, chromedp.ByQuery),
		chromedp.WaitVisible(item, chromedp.ByQuery),
		chromedp.Click(item, chromedp.ByQuery),
	}
}

func readTheme(theme *string) chromedp.Action {
	return chromedp.Evaluate(`document.documentElement.getAttribute('data-bs-theme') || ''`, theme)
}

func TestColorMode_ChoiceIsSavedAndSurvivesReload(t *testing.T) {
	ts := newColorModeServer(t)
	defer ts.Close()

	ctx := newColorModeContext(t)

	var initial, picked, stored, reloaded string

	err := chromedp.Run(ctx,
		systemColorScheme("light"),
		chromedp.Navigate(ts.URL),
		chromedp.WaitReady(`#color-mode-toggle`, chromedp.ByQuery),
		readTheme(&initial),
		pickColorMode("dark"),
		readTheme(&picked),
		chromedp.Evaluate(`localStorage.getItem('lte-theme') || ''`, &stored),
		chromedp.Reload(),
		chromedp.WaitReady(`#color-mode-toggle`, chromedp.ByQuery),
		readTheme(&reloaded),
	)
	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if initial != "light" {
		t.Errorf("with a light system and no saved choice, theme = %q, want light", initial)
	}

	if picked != "dark" || stored != "dark" {
		t.Errorf("after picking Dark, theme = %q and saved = %q, want dark and dark", picked, stored)
	}

	if reloaded != "dark" {
		t.Errorf("after reload, theme = %q, want dark", reloaded)
	}
}

func TestColorMode_AutoFollowsSystem(t *testing.T) {
	ts := newColorModeServer(t)
	defer ts.Close()

	ctx := newColorModeContext(t)

	var whenDark, whenLight string

	err := chromedp.Run(ctx,
		systemColorScheme("dark"),
		chromedp.Navigate(ts.URL),
		chromedp.WaitReady(`#color-mode-toggle`, chromedp.ByQuery),
		pickColorMode("auto"),
		readTheme(&whenDark),
		systemColorScheme("light"),
		chromedp.Sleep(200*time.Millisecond),
		readTheme(&whenLight),
	)
	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if whenDark != "dark" || whenLight != "light" {
		t.Errorf("in Auto, theme = %q on a dark system and %q on a light one, want dark and light", whenDark, whenLight)
	}
}

func TestThemeInit_AppliesSavedModeOnItsOwn(t *testing.T) {
	ts := newColorModeServer(t)
	defer ts.Close()

	ctx := newColorModeContext(t)

	var theme string

	err := chromedp.Run(ctx,
		systemColorScheme("light"),
		chromedp.Navigate(ts.URL+"/init-only"),
		chromedp.Evaluate(`localStorage.setItem('lte-theme', 'dark')`, nil),
		chromedp.Reload(),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		readTheme(&theme),
	)
	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if theme != "dark" {
		t.Errorf("theme-init.js alone set theme %q, want dark", theme)
	}
}
