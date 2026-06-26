package branding

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

// makePNG encodes a w×h PNG and returns its bytes.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	return buf.Bytes()
}

func TestValidateSquareFavicon_PNG(t *testing.T) {
	if err := validateSquareFavicon(makePNG(t, 32, 32), kindPNG); err != nil {
		t.Errorf("square PNG rejected: %v", err)
	}

	if err := validateSquareFavicon(makePNG(t, 64, 32), kindPNG); err == nil {
		t.Error("non-square PNG accepted, want rejection")
	}

	if err := validateSquareFavicon([]byte("not a png"), kindPNG); err == nil {
		t.Error("invalid PNG accepted, want rejection")
	}
}

func TestValidateSquareFavicon_SVG(t *testing.T) {
	tests := []struct {
		name    string
		svg     string
		wantErr bool
	}{
		{"square viewBox", `<svg viewBox="0 0 24 24"></svg>`, false},
		{"non-square viewBox", `<svg viewBox="0 0 48 24"></svg>`, true},
		{"square width/height", `<svg width="32" height="32"></svg>`, false},
		{"non-square width/height", `<svg width="32px" height="16px"></svg>`, true},
		{"no dimensions accepted", `<svg></svg>`, false},
		{"stroke-width ignored", `<svg viewBox="0 0 10 10" stroke-width="4"></svg>`, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSquareFavicon([]byte(tc.svg), kindSVG)
			if tc.wantErr && err == nil {
				t.Error("expected rejection, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("expected acceptance, got %v", err)
			}
		})
	}
}

func TestValidateSquareFavicon_LogoUnconstrained(t *testing.T) {
	// A non-square logo must be accepted (no aspect-ratio constraint).
	if err := validateSquareFavicon(makePNG(t, 200, 50), kindImage); err != nil {
		t.Errorf("non-square logo rejected: %v", err)
	}
}
