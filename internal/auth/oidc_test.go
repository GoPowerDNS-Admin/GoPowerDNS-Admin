package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// --- fake OIDC server ---

// fakeOIDCServer is a httptest.Server that speaks just enough OIDC to exercise
// OIDCProvider without hitting a real identity provider.
type fakeOIDCServer struct {
	*httptest.Server
	key *rsa.PrivateKey

	mu          sync.Mutex
	tokenError  bool                   // /token returns HTTP 400
	noIDToken   bool                   // /token omits id_token field
	withLogout  bool                   // discovery includes end_session_endpoint
	extraClaims map[string]interface{} // token payload overrides (nil = defaults)
}

func newFakeOIDCServer(t *testing.T) *fakeOIDCServer {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	f := &fakeOIDCServer{key: key}

	mux := http.NewServeMux()
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		withLogout := f.withLogout
		f.mu.Unlock()

		doc := map[string]interface{}{
			"issuer":                                f.URL,
			"authorization_endpoint":                f.URL + "/auth",
			"token_endpoint":                        f.URL + "/token",
			"jwks_uri":                              f.URL + "/keys",
			"userinfo_endpoint":                     f.URL + "/userinfo",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		if withLogout {
			doc["end_session_endpoint"] = f.URL + "/logout"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})

	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		pub := &f.key.PublicKey
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": "test-key",
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		tokenError := f.tokenError
		noIDToken := f.noIDToken
		extra := f.extraClaims
		f.mu.Unlock()

		if tokenError {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"bad code"}`))

			return
		}

		resp := map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		}
		if !noIDToken {
			resp["id_token"] = f.issueToken(t, extra)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":            "user123",
			"email":          "test@example.com",
			"name":           "Test User",
			"email_verified": true,
		})
	})

	return f
}

// issueToken produces a signed RS256 JWT. extra overrides/extends the default
// claims (email, name, groups …). Pass nil for defaults.
func (f *fakeOIDCServer) issueToken(t *testing.T, extra map[string]interface{}) string {
	t.Helper()

	now := time.Now()

	payload := map[string]interface{}{
		"iss": f.URL,
		"sub": "user123",
		"aud": []string{"test-client-id"},
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		// default user claims
		"email":          "test@example.com",
		"email_verified": true,
		"name":           "Test User",
		"given_name":     "Test",
		"family_name":    "User",
		"groups":         []string{"admin", "users"},
	}
	for k, v := range extra {
		payload[k] = v
	}

	return signRS256JWT(t, f.key, "test-key", payload)
}

// signRS256JWT builds a compact RS256 JWT from an arbitrary payload map.
func signRS256JWT(t *testing.T, key *rsa.PrivateKey, kid string, payload map[string]interface{}) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	})
	require.NoError(t, err)

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	sigInput := fmt.Sprintf("%s.%s",
		base64.RawURLEncoding.EncodeToString(header),
		base64.RawURLEncoding.EncodeToString(body),
	)

	h := sha256.New()
	h.Write([]byte(sigInput))

	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	require.NoError(t, err)

	return sigInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// --- helpers ---

func newOIDCTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))

	return db
}

func newTestOIDCProvider(t *testing.T, srv *fakeOIDCServer, db *gorm.DB, groupsClaim string) *OIDCProvider {
	t.Helper()

	cfg := &OIDCConfig{
		Enabled:      true,
		ProviderURL:  srv.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost/callback",
		GroupsClaim:  groupsClaim,
	}

	p, err := NewOIDCProvider(context.Background(), cfg, db)
	require.NoError(t, err)

	return p
}

// --- tests ---

func TestGenerateStateToken(t *testing.T) {
	token, err := GenerateStateToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	decoded, err := base64.URLEncoding.DecodeString(token)
	require.NoError(t, err)
	assert.Len(t, decoded, 32)

	// Each call must produce a distinct value.
	token2, err := GenerateStateToken()
	require.NoError(t, err)
	assert.NotEqual(t, token, token2)
}

func TestNewOIDCProvider_Disabled(t *testing.T) {
	p, err := NewOIDCProvider(context.Background(), &OIDCConfig{Enabled: false}, nil)
	require.ErrorIs(t, err, ErrOIDCDisabled)
	assert.Nil(t, p)
}

func TestNewOIDCProvider(t *testing.T) {
	srv := newFakeOIDCServer(t)
	p := newTestOIDCProvider(t, srv, newOIDCTestDB(t), "")
	assert.NotNil(t, p)
}

func TestGetAuthURL(t *testing.T) {
	srv := newFakeOIDCServer(t)
	p := newTestOIDCProvider(t, srv, newOIDCTestDB(t), "")

	url := p.GetAuthURL("csrf-state-value")
	assert.Contains(t, url, "/auth")
	assert.Contains(t, url, "state=csrf-state-value")
	assert.Contains(t, url, "client_id=test-client-id")
}

func TestHandleCallback(t *testing.T) {
	tests := []struct {
		name        string
		tokenError  bool
		noIDToken   bool
		extraClaims map[string]interface{}
		seedUser    *models.User // pre-existing DB record
		wantErr     bool
		wantErrIs   error
		checkUser   func(t *testing.T, u *models.User)
		wantGroups  []string
	}{
		{
			name: "creates new user from token claims",
			checkUser: func(t *testing.T, u *models.User) {
				t.Helper()
				assert.Equal(t, "test@example.com", u.Username)
				assert.Equal(t, "test@example.com", u.Email)
				assert.Equal(t, "Test", u.FirstName)
				assert.Equal(t, "User", u.LastName)
				assert.Equal(t, models.AuthSourceOIDC, u.AuthSource)
				assert.Equal(t, "user123", u.ExternalID)
				assert.True(t, u.Active)
			},
			wantGroups: []string{"admin", "users"},
		},
		{
			name: "updates existing user",
			seedUser: &models.User{
				Active:     true,
				Username:   "old@example.com",
				Email:      "old@example.com",
				FirstName:  "Old",
				LastName:   "Name",
				AuthSource: models.AuthSourceOIDC,
				ExternalID: "user123",
			},
			checkUser: func(t *testing.T, u *models.User) {
				t.Helper()
				assert.Equal(t, "test@example.com", u.Email)
				assert.Equal(t, "Test", u.FirstName)
				assert.Equal(t, "User", u.LastName)
			},
		},
		{
			name:       "token endpoint error",
			tokenError: true,
			wantErr:    true,
		},
		{
			name:      "no id_token in response",
			noIDToken: true,
			wantErr:   true,
			wantErrIs: ErrNoIDToken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newFakeOIDCServer(t)
			db := newOIDCTestDB(t)

			srv.mu.Lock()
			srv.tokenError = tc.tokenError
			srv.noIDToken = tc.noIDToken
			srv.extraClaims = tc.extraClaims
			srv.mu.Unlock()

			if tc.seedUser != nil {
				require.NoError(t, db.Create(tc.seedUser).Error)
			}

			p := newTestOIDCProvider(t, srv, db, "")

			user, groups, err := p.HandleCallback(context.Background(), "auth-code")

			if tc.wantErr {
				require.Error(t, err)

				if tc.wantErrIs != nil {
					require.ErrorIs(t, err, tc.wantErrIs)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, user)

			if tc.checkUser != nil {
				tc.checkUser(t, user)
			}

			if tc.wantGroups != nil {
				assert.Equal(t, tc.wantGroups, groups)
			}
		})
	}
}

func TestVerifyToken(t *testing.T) {
	srv := newFakeOIDCServer(t)
	p := newTestOIDCProvider(t, srv, newOIDCTestDB(t), "")

	t.Run("valid token", func(t *testing.T) {
		raw := srv.issueToken(t, nil)
		idToken, err := p.VerifyToken(context.Background(), raw)
		require.NoError(t, err)
		assert.Equal(t, "user123", idToken.Subject)
	})

	t.Run("garbage string", func(t *testing.T) {
		_, err := p.VerifyToken(context.Background(), "not.a.jwt")
		require.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		past := time.Now().Add(-2 * time.Hour)
		raw := signRS256JWT(t, srv.key, "test-key", map[string]interface{}{
			"iss": srv.URL,
			"sub": "user123",
			"aud": []string{"test-client-id"},
			"exp": past.Unix(),
			"iat": past.Add(-time.Hour).Unix(),
		})
		_, err := p.VerifyToken(context.Background(), raw)
		require.Error(t, err)
	})
}

func TestGroupsFromToken(t *testing.T) {
	srv := newFakeOIDCServer(t)
	db := newOIDCTestDB(t)

	tests := []struct {
		name          string
		groupsClaim   string
		tokenClaims   map[string]interface{}
		defaultGroups []string
		want          []string
	}{
		{
			name:          "empty GroupsClaim returns default groups",
			groupsClaim:   "",
			tokenClaims:   map[string]interface{}{"email": "u@example.com"},
			defaultGroups: []string{"a", "b"},
			want:          []string{"a", "b"},
		},
		{
			name:          "GroupsClaim=groups returns default groups",
			groupsClaim:   "groups",
			tokenClaims:   map[string]interface{}{"email": "u@example.com"},
			defaultGroups: []string{"x"},
			want:          []string{"x"},
		},
		{
			name:        "custom claim present in token",
			groupsClaim: "roles",
			tokenClaims: map[string]interface{}{
				"email": "u@example.com",
				"roles": []string{"admin", "viewer"},
			},
			want: []string{"admin", "viewer"},
		},
		{
			name:          "custom claim absent falls back to default",
			groupsClaim:   "roles",
			tokenClaims:   map[string]interface{}{"email": "u@example.com"},
			defaultGroups: []string{"fallback"},
			want:          []string{"fallback"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newTestOIDCProvider(t, srv, db, tc.groupsClaim)

			raw := srv.issueToken(t, tc.tokenClaims)
			idToken, err := p.VerifyToken(context.Background(), raw)
			require.NoError(t, err)

			got := p.groupsFromToken(idToken, tc.defaultGroups)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetLogoutURL(t *testing.T) {
	t.Run("provider supports end_session_endpoint", func(t *testing.T) {
		srv := newFakeOIDCServer(t)
		srv.withLogout = true // must be set before NewOIDCProvider fetches the discovery doc
		p := newTestOIDCProvider(t, srv, newOIDCTestDB(t), "")

		url := p.GetLogoutURL("raw-id-token", "http://localhost/after-logout")
		assert.Contains(t, url, "/logout")
		assert.Contains(t, url, "id_token_hint=raw-id-token")
		assert.Contains(t, url, "post_logout_redirect_uri=")
	})

	t.Run("provider does not support end_session_endpoint", func(t *testing.T) {
		srv := newFakeOIDCServer(t)
		p := newTestOIDCProvider(t, srv, newOIDCTestDB(t), "")

		url := p.GetLogoutURL("raw-id-token", "http://localhost/after-logout")
		assert.Empty(t, url)
	})
}

func TestGetUserInfo(t *testing.T) {
	srv := newFakeOIDCServer(t)
	p := newTestOIDCProvider(t, srv, newOIDCTestDB(t), "")

	claims, err := p.GetUserInfo(context.Background(), "test-access-token")
	require.NoError(t, err)

	assert.Equal(t, "user123", claims["sub"])
	assert.Equal(t, "test@example.com", claims["email"])
}
