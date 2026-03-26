//go:build browser

package web_test

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// allocatorOpts returns chromedp allocator options, adding --no-sandbox when
// running inside CI (where the kernel sandbox is typically unavailable).
func allocatorOpts() []func(*chromedp.ExecAllocator) {
	opts := chromedp.DefaultExecAllocatorOptions[:]
	if os.Getenv("CI") != "" {
		opts = append(opts, chromedp.NoSandbox)
	}

	return opts
}

// zoneEditorTestPage is a minimal self-contained page that loads Alpine +
// zone-edit.js and renders both the SOA and record modals, mirroring the
// real edit.gohtml structure closely enough to exercise the JS logic.
const zoneEditorTestPage = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>

<!-- Init data read by zoneEditor() -->
<script type="application/json" id="zone-data">
{
  "zoneName": "example.com.",
  "allowedTypes": [
    {"type": "A",    "description": "IPv4 Address"},
    {"type": "AAAA", "description": "IPv6 Address"},
    {"type": "SOA",  "description": ""}
  ],
  "records": [{
    "name": "example.com.",
    "type": "SOA",
    "ttl":  300,
    "content": "ns1.example.com. hostmaster.example.com. 2023010101 3600 900 604800 300",
    "disabled": false,
    "comment": ""
  }],
  "ttlPresets": [
    {"seconds": 300,  "label": "5 min"},
    {"seconds": 3600, "label": "1 hour"}
  ],
  "pageSize": 25
}
</script>

<div x-data="zoneEditor()" id="zone-editor">
  <div id="toast-container"></div>

  <!-- Test triggers -->
  <button id="open-soa-btn"
          @click="openSOAModal(records.find(r => r.type === 'SOA'))">
    Edit SOA
  </button>
  <button id="open-add-record-btn" @click="openAddRecord()">Add Record</button>

  <!-- SOA Modal — mirrors edit.gohtml -->
  <div class="modal fade" id="soaModal" tabindex="-1" aria-labelledby="soaModalLabel" aria-hidden="true">
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title" id="soaModalLabel">Edit SOA Record</h5>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
        </div>
        <div class="modal-body">
          <form id="soa-form">
            <input type="text"   id="soa-mname"   x-model="soaForm.mname"   required>
            <input type="text"   id="soa-rname"   x-model="soaForm.rname"   required>
            <input type="number" id="soa-serial"  x-model="soaForm.serial"  required min="1">
            <input type="number" id="soa-refresh" x-model="soaForm.refresh" required min="1">
            <input type="number" id="soa-retry"   x-model="soaForm.retry"   required min="1">
            <input type="number" id="soa-expire"  x-model="soaForm.expire"  required min="1">
            <input type="number" id="soa-minimum" x-model="soaForm.minimum" required min="0">
          </form>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
          <button type="button" id="save-soa-btn" @click="saveSOA()">Save SOA</button>
        </div>
      </div>
    </div>
  </div>

  <!-- Record Modal — mirrors edit.gohtml -->
  <div class="modal fade" id="recordModal" tabindex="-1" aria-labelledby="recordModalLabel">
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title" id="recordModalLabel"
              x-text="recordForm.isEditing ? 'Edit Record' : 'Add Record'"></h5>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
        </div>
        <div class="modal-body">
          <form id="record-form">
            <input type="text" id="record-name-input"
                   x-model="recordForm.name" required>
            <select id="record-type-input"
                    x-model="recordForm.type"
                    x-show="!recordForm.isEditing" required>
              <option value="A">A</option>
              <option value="AAAA">AAAA</option>
            </select>
            <input type="number" id="record-ttl-input"
                   x-model.number="recordForm.ttl" required min="1">
            <input type="text" id="record-content-input"
                   x-model="recordForm.content"
                   :required="recordForm.type !== 'MX' && recordForm.type !== 'TXT'">
            <textarea id="record-comment-input"
                      x-model="recordForm.comment" maxlength="255"></textarea>
            <input type="checkbox" id="record-disabled-input"
                   x-model="recordForm.disabled">
          </form>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
          <button type="button" id="save-record-btn" @click="saveRecord()">
            <span x-text="recordForm.isEditing ? 'Update Record' : 'Add Record'"></span>
          </button>
        </div>
      </div>
    </div>
  </div>
</div>

<script src="/static/vendor/bootstrap-5.3.8-dist/js/bootstrap.bundle.min.js"></script>
<script src="/static/js/zone-edit.js"></script>
<script src="/static/vendor/alpinejs-3.14.9/alpine.min.js" defer></script>
</body>
</html>`

// newTestServer starts an HTTP server that serves the SOA test page and the
// project's embedded static assets (Alpine, Bootstrap, zone-edit.js).
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	staticFS, err := fs.Sub(os.DirFS("static"), ".")
	if err != nil {
		t.Fatalf("open static dir: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(zoneEditorTestPage))
	})

	return httptest.NewServer(mux)
}

// TestSOAModal_SaveWithoutTypeError verifies that interacting with the SOA
// serial input (a number field) and then clicking Save does not throw a
// TypeError. Regression test for the x-model.number coercion bug where
// Alpine coerced soaForm.serial to a JS number, causing .trim() to fail.
func TestSOAModal_SaveWithoutTypeError(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts()...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var dangerToasts int
	var jsErrors []string

	err := chromedp.Run(ctx,
		// 1. Load the test page.
		chromedp.Navigate(ts.URL),

		// 2. Wait for Alpine to boot and bind the component.
		chromedp.WaitVisible(`#open-soa-btn`),

		// 3. Open the SOA modal.
		chromedp.Click(`#open-soa-btn`, chromedp.ByID),

		// 4. Wait for Bootstrap to animate the modal in.
		chromedp.WaitVisible(`#soaModal.show`, chromedp.ByQuery),

		// 5. Interact with the serial field to trigger Alpine's x-model sync.
		//    This is the step that coerced soaForm.serial to a JS number with the
		//    old x-model.number binding, causing saveSOA to throw a TypeError.
		//    We set the value and dispatch an input event, which is what Alpine
		//    listens to for x-model updates.
		chromedp.Evaluate(`
			const el = document.getElementById('soa-serial');
			el.value = '2023010102';
			el.dispatchEvent(new Event('input', { bubbles: true }));
		`, nil),

		// 6. Click Save.
		chromedp.Click(`#save-soa-btn`, chromedp.ByID),

		// 7. Wait for Bootstrap to finish hiding the modal.
		chromedp.WaitNotVisible(`#soaModal`, chromedp.ByID),

		// 8. Assert no danger toasts (a TypeError from saveSOA would produce one).
		chromedp.Evaluate(
			`document.querySelectorAll('#toast-container .toast-header.bg-danger').length`,
			&dangerToasts,
		),
	)

	if err != nil {
		t.Fatalf("chromedp run failed: %v\nJS errors: %v", err, jsErrors)
	}

	if dangerToasts > 0 {
		t.Errorf("expected 0 danger toasts (TypeError would produce one), got %d", dangerToasts)
	}
}

// TestSOAModal_SerialIsStoredAfterSave verifies that after editing the SOA
// serial and clicking Save, the updated serial is reflected in Alpine's
// records state (i.e. the in-memory record is actually mutated).
func TestSOAModal_SerialIsStoredAfterSave(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts()...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	const newSerial = "2023010102"

	var soaContent string

	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible(`#open-soa-btn`),
		chromedp.Click(`#open-soa-btn`, chromedp.ByID),
		chromedp.WaitVisible(`#soaModal.show`, chromedp.ByQuery),

		// Update the serial field and trigger Alpine's x-model sync.
		chromedp.Evaluate(`
			const el = document.getElementById('soa-serial');
			el.value = '`+newSerial+`';
			el.dispatchEvent(new Event('input', { bubbles: true }));
		`, nil),

		chromedp.Click(`#save-soa-btn`, chromedp.ByID),
		chromedp.WaitNotVisible(`#soaModal`, chromedp.ByID),

		// Read the SOA record content directly from Alpine's records array.
		chromedp.Evaluate(`
			Alpine.$data(document.getElementById('zone-editor'))
				.records.find(r => r.type === 'SOA').content
		`, &soaContent),
	)

	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if soaContent == "" {
		t.Fatal("SOA record content is empty after save")
	}

	// The composed SOA content must contain the new serial.
	if !strings.Contains(soaContent, newSerial) {
		t.Errorf("expected SOA content to contain serial %q, got: %q", newSerial, soaContent)
	}
}

// TestRecord_AddIPv4WithComment verifies that a new A record with an IPv4
// address and a comment can be added via the record modal, and that both
// the content and comment are stored in Alpine's records state.
func TestRecord_AddIPv4WithComment(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts()...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	const (
		recordName    = "www"
		recordIP      = "192.168.1.100"
		recordComment = "test IPv4 record"
	)

	var recordResult map[string]interface{}

	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible(`#open-add-record-btn`),

		// Open the Add Record modal.
		chromedp.Click(`#open-add-record-btn`, chromedp.ByID),
		chromedp.WaitVisible(`#recordModal.show`, chromedp.ByQuery),

		// Type "A" is the default. Set name, content and comment, firing
		// input events so Alpine's x-model picks up each change.
		chromedp.Evaluate(`
			['record-name-input', 'record-content-input', 'record-comment-input'].forEach((id, i) => {
				const el = document.getElementById(id);
				el.value = ['`+recordName+`', '`+recordIP+`', '`+recordComment+`'][i];
				el.dispatchEvent(new Event('input', { bubbles: true }));
			});
		`, nil),

		// Save the record.
		chromedp.Click(`#save-record-btn`, chromedp.ByID),
		chromedp.WaitNotVisible(`#recordModal`, chromedp.ByID),

		// Read content and comment of the newly added A record in one call.
		chromedp.Evaluate(`
			(() => {
				const rec = Alpine.$data(document.getElementById('zone-editor'))
					.records.find(r => r.type === 'A');
				return rec ? { content: rec.content, comment: rec.comment } : {};
			})()
		`, &recordResult),
	)

	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if got, ok := recordResult["content"]; !ok || got != recordIP {
		t.Errorf("expected A record content %q, got %q", recordIP, got)
	}

	if got, ok := recordResult["comment"]; !ok || got != recordComment {
		t.Errorf("expected A record comment %q, got %q", recordComment, got)
	}
}

// TestRecord_AddIPv6WithComment verifies that a new AAAA record with an IPv6
// address and a comment can be added via the record modal, and that both
// the content and comment are stored in Alpine's records state.
func TestRecord_AddIPv6WithComment(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts()...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	const (
		recordName    = "ipv6"
		recordIP      = "2001:db8::1"
		recordComment = "test IPv6 record"
	)

	var recordResult map[string]interface{}

	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible(`#open-add-record-btn`),

		// Open the Add Record modal.
		chromedp.Click(`#open-add-record-btn`, chromedp.ByID),
		chromedp.WaitVisible(`#recordModal.show`, chromedp.ByQuery),

		// Switch type to AAAA, then fill name, content and comment.
		chromedp.Evaluate(`
			(() => {
				const sel = document.getElementById('record-type-input');
				sel.value = 'AAAA';
				sel.dispatchEvent(new Event('change', { bubbles: true }));

				[['record-name-input', '`+recordName+`'],
				 ['record-content-input', '`+recordIP+`'],
				 ['record-comment-input', '`+recordComment+`']].forEach(([id, val]) => {
					const el = document.getElementById(id);
					el.value = val;
					el.dispatchEvent(new Event('input', { bubbles: true }));
				});
			})()
		`, nil),

		// Save the record.
		chromedp.Click(`#save-record-btn`, chromedp.ByID),
		chromedp.WaitNotVisible(`#recordModal`, chromedp.ByID),

		// Read content and comment of the newly added AAAA record in one call.
		chromedp.Evaluate(`
			(() => {
				const rec = Alpine.$data(document.getElementById('zone-editor'))
					.records.find(r => r.type === 'AAAA');
				return rec ? { content: rec.content, comment: rec.comment } : {};
			})()
		`, &recordResult),
	)

	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}

	if got, ok := recordResult["content"]; !ok || got != recordIP {
		t.Errorf("expected AAAA record content %q, got %q", recordIP, got)
	}

	if got, ok := recordResult["comment"]; !ok || got != recordComment {
		t.Errorf("expected AAAA record comment %q, got %q", recordComment, got)
	}
}
