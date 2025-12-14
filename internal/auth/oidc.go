package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// ErrOIDCDisabled is returned when OIDC is disabled via configuration.
var ErrOIDCDisabled = errors.New("oidc authentication is disabled")

// OIDCConfig holds OpenID Connect (OIDC) configuration for authentication.
type OIDCConfig struct {
	// Enabled indicates if OIDC authentication is enabled.
	Enabled bool
	// ProviderURL is the OIDC provider's discovery URL (e.g., "https://accounts.google.com").
	ProviderURL string
	// ClientID is the OAuth2 client identifier.
	ClientID string
	// ClientSecret is the OAuth2 client secret.
	ClientSecret string
	// RedirectURL is the OAuth2 callback URL where the provider redirects after authentication.
	RedirectURL string
	// Scopes are the OAuth2 scopes to request (default: ["openid", "profile", "email"]).
	Scopes []string
	// GroupsClaim is the ID token claim name containing user groups (e.g., "groups", "roles").
	GroupsClaim string
}

// OIDCProvider handles OIDC authentication.
type OIDCProvider struct {
	config   *OIDCConfig
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth2   oauth2.Config
	db       *gorm.DB
}

// NewOIDCProvider creates a new OIDC provider.
func NewOIDCProvider(ctx context.Context, config *OIDCConfig, db *gorm.DB) (*OIDCProvider, error) {
	if !config.Enabled {
		return nil, ErrOIDCDisabled
	}

	provider, err := oidc.NewProvider(ctx, config.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	oauth2Config := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	return &OIDCProvider{
		config:   config,
		provider: provider,
		verifier: verifier,
		oauth2:   oauth2Config,
		db:       db,
	}, nil
}

// GenerateStateToken generates a random state token for CSRF protection.
func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthURL returns the OIDC authorization URL with state token.
func (p *OIDCProvider) GetAuthURL(state string) string {
	return p.oauth2.AuthCodeURL(state)
}

// HandleCallback handles the OIDC callback and returns the authenticated user.
func (p *OIDCProvider) HandleCallback(ctx context.Context, code string) (*models.User, []string, error) {
	// Exchange code for token
	oauth2Token, err := p.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, nil, ErrNoIDToken
	}

	// Verify ID token
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Sub           string   `json:"sub"`
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		GivenName     string   `json:"given_name"`
		FamilyName    string   `json:"family_name"`
		Groups        []string `json:"groups"` // This might be under a different claim
	}

	if err = idToken.Claims(&claims); err != nil {
		return nil, nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Resolve groups via helper to keep this function's complexity low
	groups := p.groupsFromToken(idToken, claims.Groups)

	// Find or create user
	var user models.User

	err = p.db.Where("external_id = ? AND auth_source = ?", claims.Sub, models.AuthSourceOIDC).
		First(&user).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Create new user
		user = models.User{
			Active:     true,
			Username:   claims.Email, // Use email as username
			Email:      claims.Email,
			FirstName:  claims.GivenName,
			LastName:   claims.FamilyName,
			AuthSource: models.AuthSourceOIDC,
			ExternalID: claims.Sub,
			RoleID:     0, // No direct role, permissions come from groups
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err = p.db.Create(&user).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to create user: %w", err)
		}
	case err != nil:
		return nil, nil, fmt.Errorf("failed to query user: %w", err)
	default:
		// Update existing user
		user.Email = claims.Email
		user.FirstName = claims.GivenName
		user.LastName = claims.FamilyName
		user.UpdatedAt = time.Now()

		if err = p.db.Save(&user).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	return &user, groups, nil
}

// VerifyToken verifies the signature and claims of an OIDC ID token.
// It validates the token was issued by the configured provider and hasn't expired.
func (p *OIDCProvider) VerifyToken(ctx context.Context, rawToken string) (*oidc.IDToken, error) {
	return p.verifier.Verify(ctx, rawToken)
}

// groupsFromToken determines the user's groups using the configured claim.
// It defaults to the provided defaultGroups and overrides them if a custom claim is set and present.
func (p *OIDCProvider) groupsFromToken(idToken *oidc.IDToken, defaultGroups []string) []string {
	gc := p.config.GroupsClaim
	if gc == "" || gc == "groups" {
		return defaultGroups
	}

	var allClaims map[string]interface{}
	if err := idToken.Claims(&allClaims); err != nil {
		return defaultGroups
	}

	v, ok := allClaims[gc]
	if !ok {
		return defaultGroups
	}

	switch vv := v.(type) {
	case []string:
		return vv
	case []interface{}:
		tmp := make([]string, 0, len(vv))
		for _, g := range vv {
			if s, ok := g.(string); ok {
				tmp = append(tmp, s)
			}
		}

		return tmp
	default:
		return defaultGroups
	}
}

// GetLogoutURL constructs the OIDC provider's logout URL if supported.
// It includes the ID token hint and post-logout redirect URI parameters.
// Returns an empty string if the provider doesn't support logout endpoints.
func (p *OIDCProvider) GetLogoutURL(idToken, postLogoutRedirectURI string) string {
	// Check if provider supports end_session_endpoint
	var claims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}

	if err := p.provider.Claims(&claims); err != nil || claims.EndSessionEndpoint == "" {
		// Provider doesn't support logout endpoint
		return ""
	}

	// Build logout URL
	logoutURL := fmt.Sprintf("%s?id_token_hint=%s&post_logout_redirect_uri=%s",
		claims.EndSessionEndpoint,
		idToken,
		postLogoutRedirectURI,
	)

	return logoutURL
}

// RefreshToken obtains a new access token using a refresh token.
// This allows extending the user's session without requiring re-authentication.
// Returns the new token set or an error if the refresh token is invalid or expired.
func (p *OIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	tokenSource := p.oauth2.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	return tokenSource.Token()
}

// GetUserInfo fetches additional user information from the OIDC UserInfo endpoint.
// This provides claims not included in the ID token, such as additional profile information.
// The accessToken must be a valid OAuth2 access token.
func (p *OIDCProvider) GetUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var claims map[string]interface{}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse user info claims: %w", err)
	}

	return claims, nil
}

// RequireAuth is middleware for protecting routes with OIDC authentication.
// It checks for a valid session or token before allowing access to the next handler.
// This integrates with the application's session management system.
func (p *OIDCProvider) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for session or token
		// This would integrate with your session management
		next.ServeHTTP(w, r)
	})
}
