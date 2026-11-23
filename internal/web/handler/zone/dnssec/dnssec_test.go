package dnssec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pdnsapi "github.com/joeig/go-powerdns/v3"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

// newFakePowerDNS serves the cryptokey create call and answers the rectify
// call with rectifyStatus. Every request is reported on the returned channel
// as "<method> <path>".
func newFakePowerDNS(t *testing.T, rectifyStatus int) <-chan string {
	t.Helper()

	calls := make(chan string, 2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls <- r.Method + " " + r.URL.Path

		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1,"keytype":"csk","active":true,"published":true}`))

			return
		}

		w.WriteHeader(rectifyStatus)
		_, _ = w.Write([]byte(`{"result":"Rectified"}`))
	}))

	powerdns.Engine.Client = pdnsapi.New(srv.URL, "localhost")

	t.Cleanup(func() {
		powerdns.Engine.Client = nil

		srv.Close()
	})

	return calls
}

func TestEnableDNSSEC_RectifiesAfterCreatingKey(t *testing.T) {
	calls := newFakePowerDNS(t, http.StatusOK)

	key, warning, err := enableDNSSEC(context.Background(), "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key == nil {
		t.Fatal("expected the created key, got nil")
	}

	if warning != "" {
		t.Errorf("expected no warning, got %q", warning)
	}

	want := []string{
		"POST /api/v1/servers/localhost/zones/example.com./cryptokeys",
		"PUT /api/v1/servers/localhost/zones/example.com./rectify",
	}

	for _, w := range want {
		if got := <-calls; got != w {
			t.Errorf("expected call %q, got %q", w, got)
		}
	}
}

func TestEnableDNSSEC_RectifyFailureIsAWarning(t *testing.T) {
	newFakePowerDNS(t, http.StatusInternalServerError)

	key, warning, err := enableDNSSEC(context.Background(), "example.com.")
	if err != nil {
		t.Fatalf("a failed rectify must not fail the enable, got %v", err)
	}

	if key == nil {
		t.Fatal("expected the created key, got nil")
	}

	if !strings.Contains(warning, "pdnsutil rectify-zone example.com.") {
		t.Errorf("expected a warning with the manual fix, got %q", warning)
	}
}
