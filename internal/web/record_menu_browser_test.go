//go:build browser

package web_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// maincssStyle extracts the inline <style>…</style> block from the real
// maincss.gohtml partial so the browser test exercises the actual page CSS
// (sticky records header + the per-row dropdown z-index lift) and stays in
// sync with it — rather than duplicating the rules here.
func maincssStyle(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile("templates/partials/maincss.gohtml")
	if err != nil {
		t.Fatalf("read maincss.gohtml: %v", err)
	}

	s := string(raw)
	start := strings.Index(s, "<style>")
	end := strings.Index(s, "</style>")
	if start < 0 || end < 0 || end < start {
		t.Fatalf("could not locate <style> block in maincss.gohtml")
	}

	return s[start : end+len("</style>")]
}

// recordMenuInitData builds the zone-data JSON for a zone with enough records
// that the records list scrolls, plus a uniquely-named "webmail" record. A
// search for "webmail" narrows the list to a single row, which shrinks the
// .table-responsive scroll box below the height of the open action menu — the
// condition that used to clip the menu (Popper can fit it neither above nor
// below inside the box, so it overflows and gets clipped without the fix).
func recordMenuInitData(t *testing.T) string {
	t.Helper()

	type rec struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Type        string `json:"type"`
		TTL         int    `json:"ttl"`
		Content     string `json:"content"`
		Disabled    bool   `json:"disabled"`
		Comment     string `json:"comment"`
	}

	var records []rec
	for i := 1; i <= 30; i++ {
		name := fmt.Sprintf("host%02d", i)
		records = append(records, rec{
			Name: name + ".example.com.", DisplayName: name, Type: "A",
			TTL: 3600, Content: fmt.Sprintf("192.0.2.%d", i),
		})
	}
	for _, m := range []struct{ name, ip string }{
		{"mail", "192.0.2.200"}, {"webmail", "192.0.2.202"}, {"smtp", "192.0.2.201"},
	} {
		records = append(records, rec{
			Name: m.name + ".example.com.", DisplayName: m.name, Type: "A",
			TTL: 3600, Content: m.ip,
		})
	}

	data := map[string]any{
		"zoneName":     "example.com.",
		"allowedTypes": []map[string]string{{"type": "A", "description": "IPv4 Address"}},
		"records":      records,
		"ttlPresets":   []map[string]any{{"seconds": 3600, "label": "1 hour"}},
		"pageSize":     25,
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal init data: %v", err)
	}

	return string(b)
}

// recordMenuTestPage mirrors the records card from edit.gohtml: a sticky
// records header with a search box, a .table-responsive wrapping the records
// table whose rows carry the three-dots Edit/Delete dropdown, and a card
// footer below. It loads Bootstrap's CSS (for .table-responsive overflow and
// the dropdown z-index defaults) and the real maincss <style> block, so the
// clipping/stacking conditions match production.
func recordMenuTestPage(t *testing.T) string {
	t.Helper()

	return `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<link rel="stylesheet" href="/static/vendor/bootstrap-5.3.8-dist/css/bootstrap.min.css">
` + maincssStyle(t) + `
</head>
<body>

<script type="application/json" id="zone-data">` + recordMenuInitData(t) + `</script>

<div x-data="zoneEditor()" id="zone-editor">
  <div id="toast-container"></div>

  <div class="card card-success card-outline mb-4">
    <div class="card-header records-card-header d-flex flex-column gap-2">
      <div class="d-flex align-items-center flex-wrap gap-2">
        <div class="card-title mb-0 me-1">DNS Records</div>
        <span class="badge bg-secondary" x-text="filteredRecords.length"></span>
        <div class="input-group input-group-sm" style="width: 200px;">
          <input type="text" id="record-search" x-model="searchQuery" class="form-control" placeholder="Search name or data…">
        </div>
      </div>
    </div>

    <div class="card-body p-0">
      <div class="table-responsive">
        <table class="table table-striped table-hover mb-0">
          <thead>
            <tr><th>Name</th><th>Type</th><th>TTL</th><th>Data</th><th width="48"></th></tr>
          </thead>
          <tbody>
            <template x-for="record in paginatedRecords" :key="recordId(record)">
              <tr>
                <td class="record-name" x-text="record.display_name"></td>
                <td><span class="badge bg-light text-dark border" x-text="record.type"></span></td>
                <td x-text="record.ttl"></td>
                <td x-text="record.content"></td>
                <td class="text-end">
                  <div class="dropdown">
                    <button type="button" class="btn btn-sm btn-light record-menu-toggle"
                            data-bs-toggle="dropdown" aria-expanded="false" data-bs-boundary="viewport">⋮</button>
                    <ul class="dropdown-menu dropdown-menu-end">
                      <li>
                        <button class="dropdown-item edit-item" type="button" @click="openEditRecord(record)">Edit</button>
                      </li>
                      <li>
                        <button class="dropdown-item text-danger" type="button" @click="deleteRecord(record)">Delete</button>
                      </li>
                    </ul>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </div>

    <div class="card-footer d-flex align-items-center flex-wrap gap-2">
      <div class="d-flex align-items-center gap-2 ms-auto flex-wrap">
        <label class="small text-muted mb-0">Rows:</label>
        <select class="form-select form-select-sm" style="width:auto"><option>25</option></select>
      </div>
    </div>
  </div>

  <!-- Minimal record modal so openEditRecord() has a target. -->
  <div class="modal fade" id="recordModal" tabindex="-1">
    <div class="modal-dialog"><div class="modal-content">
      <div class="modal-body"><form id="record-form"></form></div>
    </div></div>
  </div>
</div>

<script src="/static/vendor/bootstrap-5.3.8-dist/js/bootstrap.bundle.min.js"></script>
<script src="/static/js/zone-edit.js"></script>
<script src="/static/vendor/alpinejs-3.14.9/alpine.min.js" defer></script>
</body>
</html>`
}

// newRecordMenuServer serves the records-card test page plus the embedded
// static assets.
func newRecordMenuServer(t *testing.T) *httptest.Server {
	t.Helper()

	staticFS, err := fs.Sub(os.DirFS("static"), ".")
	if err != nil {
		t.Fatalf("open static dir: %v", err)
	}

	page := recordMenuTestPage(t)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})

	return httptest.NewServer(mux)
}

// TestRecordMenu_EditVisibleAfterSearch is a regression test for the bug where
// searching the records list (which shrinks the .table-responsive scroll box to
// a few rows) caused the per-row Edit/Delete menu to be clipped by that box and
// painted behind the card footer / sticky header — hiding the Edit option.
//
// The fix positions the menu with Popper's `fixed` strategy (zone-edit.js) and
// lifts its z-index above the sticky header (maincss.gohtml). This test opens
// the last filtered row's menu and asserts the Edit item is the topmost element
// at its own center — i.e. a real click would land on it, not on whatever would
// otherwise occlude it.
func TestRecordMenu_EditVisibleAfterSearch(t *testing.T) {
	ts := newRecordMenuServer(t)
	defer ts.Close()

	opts := append(allocatorOpts(), chromedp.WindowSize(1280, 800))

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var result struct {
		Found         bool    `json:"found"`
		RowCount      int     `json:"rowCount"`
		TopmostIsEdit bool    `json:"topmostIsEdit"`
		TopmostDesc   string  `json:"topmostDesc"`
		EditTop       float64 `json:"editTop"`
		EditBottom    float64 `json:"editBottom"`
		BoxBottom     float64 `json:"boxBottom"`
		MenuTop       float64 `json:"menuTop"`
	}

	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),

		// Wait for Alpine to render the records rows.
		chromedp.WaitVisible(`#zone-editor tbody tr`, chromedp.ByQuery),

		// Narrow the list to a single row via the search box, shrinking the
		// scroll box below the open menu's height.
		chromedp.Evaluate(`
			(() => {
				const c = Alpine.$data(document.getElementById('zone-editor'));
				c.searchQuery = 'webmail';
			})()
		`, nil),

		// Let Alpine re-render the filtered (shorter) table.
		chromedp.Sleep(300*time.Millisecond),

		// Open the remaining row's three-dots menu. Its menu overflows the
		// now-tiny scroll box and (without the fix) is clipped by it.
		chromedp.Evaluate(`
			(() => {
				const toggles = document.querySelectorAll('#zone-editor .record-menu-toggle');
				toggles[toggles.length - 1].click();
			})()
		`, nil),

		// Wait for the menu to open.
		chromedp.WaitVisible(`#zone-editor .dropdown-menu.show`, chromedp.ByQuery),
		chromedp.Sleep(200*time.Millisecond),

		// Hit-test: is the Edit item the element actually painted at its own
		// center? If it is clipped or stacked under the footer/header, some other
		// element wins the hit-test and a user click would miss the Edit button.
		chromedp.Evaluate(`
			(() => {
				const rows = document.querySelectorAll('#zone-editor tbody tr').length;
				const edit = document.querySelector('#zone-editor .dropdown-menu.show .edit-item');
				if (!edit) return { found: false, rowCount: rows };
				const r = edit.getBoundingClientRect();
				const top = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
				const isEdit = !!(top && (top === edit || edit.contains(top) || top.contains(edit)));
				const desc = top ? (top.tagName + '.' + (top.className || '').toString().trim().split(/\s+/).join('.')) : 'null';
				const box = document.querySelector('#zone-editor .table-responsive').getBoundingClientRect();
				const menu = document.querySelector('#zone-editor .dropdown-menu.show').getBoundingClientRect();
				return { found: true, rowCount: rows, topmostIsEdit: isEdit, topmostDesc: desc,
				         editTop: r.top, editBottom: r.bottom, boxBottom: box.bottom, menuTop: menu.top };
			})()
		`, &result),
	)
	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if !result.Found {
		t.Fatalf("Edit menu item not found after opening the row menu (rows after search: %d)", result.RowCount)
	}

	if result.RowCount != 1 {
		t.Fatalf("expected search to narrow to 1 row, got %d; test setup is wrong", result.RowCount)
	}

	if !result.TopmostIsEdit {
		t.Errorf("Edit menu item is occluded after search: the element painted at its center is %q, "+
			"not the Edit button — the per-row menu is being clipped/stacked behind other content "+
			"(editTop=%.0f editBottom=%.0f scrollBoxBottom=%.0f menuTop=%.0f)",
			result.TopmostDesc, result.EditTop, result.EditBottom, result.BoxBottom, result.MenuTop)
	}
}
