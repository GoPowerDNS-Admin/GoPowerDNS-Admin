package auth

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/rs/zerolog/log"

	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
)

// ErrLDAPDisabled is returned when LDAP authentication is disabled via configuration.
var ErrLDAPDisabled = errors.New("ldap authentication is disabled")

// LDAPConfig holds LDAP/Active Directory configuration for authentication.
type LDAPConfig struct {
	// Enabled indicates if LDAP authentication is enabled.
	Enabled bool
	// Host is the LDAP server hostname or IP address.
	Host string
	// Port is the LDAP server port (typically 389 for LDAP, 636 for LDAPS).
	Port int
	// UseSSL enables LDAPS (LDAP over SSL/TLS) on port 636.
	UseSSL bool
	// UseTLS enables StartTLS to upgrade an LDAP connection to TLS.
	UseTLS bool
	// SkipVerify skips TLS certificate verification (insecure, for testing only).
	SkipVerify bool
	// BindDN is the distinguished name to bind with for performing searches.
	BindDN string
	// BindPassword is the password for the bind DN.
	BindPassword string
	// BaseDN is the base distinguished name for user searches.
	BaseDN string
	// UserFilter is the LDAP filter for finding users (e.g., "(uid={username})").
	// The {username} placeholder is replaced with the actual username.
	UserFilter string
	// GroupBaseDN is the base distinguished name for group searches.
	GroupBaseDN string
	// GroupFilter is the LDAP filter for finding groups (e.g., "(member={userdn})").
	// The {userdn} placeholder is replaced with the user's DN.
	GroupFilter string
	// GroupMemberAttr is the LDAP attribute for group membership (e.g., "member", "uniqueMember").
	GroupMemberAttr string
	// UsernameAttr is the LDAP attribute containing the username (e.g., "uid", "sAMAccountName").
	UsernameAttr string
	// EmailAttr is the LDAP attribute containing the email address (e.g., "mail").
	EmailAttr string
	// FirstNameAttr is the LDAP attribute containing the first/given name (e.g., "givenName").
	FirstNameAttr string
	// LastNameAttr is the LDAP attribute containing the last/surname (e.g., "sn").
	LastNameAttr string
	// GroupNameAttr is the LDAP attribute containing the group name (e.g., "cn").
	GroupNameAttr string
	// Timeout is the connection timeout in seconds.
	Timeout int
	// SearchAttributes are additional LDAP attributes to retrieve during searches.
	SearchAttributes []string
}

// LDAPProvider handles LDAP authentication.
type LDAPProvider struct {
	config *LDAPConfig
	db     *gorm.DB
}

// NewLDAPProvider creates a new LDAP provider.
func NewLDAPProvider(config *LDAPConfig, db *gorm.DB) (*LDAPProvider, error) {
	if !config.Enabled {
		return nil, ErrLDAPDisabled
	}

	// Set defaults
	if config.UsernameAttr == "" {
		config.UsernameAttr = "uid"
	}

	if config.EmailAttr == "" {
		config.EmailAttr = "mail"
	}

	if config.FirstNameAttr == "" {
		config.FirstNameAttr = "givenName"
	}

	if config.LastNameAttr == "" {
		config.LastNameAttr = "sn"
	}

	if config.GroupNameAttr == "" {
		config.GroupNameAttr = "cn"
	}

	if config.GroupMemberAttr == "" {
		config.GroupMemberAttr = "member"
	}

	if config.Timeout == 0 {
		config.Timeout = 10
	}

	return &LDAPProvider{
		config: config,
		db:     db,
	}, nil
}

// Connect establishes a connection to the LDAP server.
func (p *LDAPProvider) Connect() (*ldap.Conn, error) {
	// Build LDAP URL
	hostPort := net.JoinHostPort(p.config.Host, strconv.Itoa(p.config.Port))

	var ldapURL string
	if p.config.UseSSL {
		ldapURL = "ldaps://" + hostPort
	} else {
		ldapURL = "ldap://" + hostPort
	}

	// Configure TLS
	var tlsConfig *tls.Config
	if p.config.UseSSL || p.config.UseTLS {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: p.config.SkipVerify, //nolint:gosec // skipping verifying tls is ok
			ServerName:         p.config.Host,
		}
	}

	// Dial using DialURL
	conn, err := ldap.DialURL(ldapURL, ldap.DialWithTLSConfig(tlsConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	// Upgrade to TLS if requested (for non-SSL connections)
	if !p.config.UseSSL && p.config.UseTLS {
		if errStartTLS := conn.StartTLS(tlsConfig); errStartTLS != nil {
			if errClose := conn.Close(); errClose != nil {
				log.Error().Err(errClose).Msg("failed to close LDAP connection")
			}

			return nil, fmt.Errorf("failed to start TLS: %w", errStartTLS)
		}
	}

	// Set timeout
	if p.config.Timeout > 0 {
		conn.SetTimeout(time.Duration(p.config.Timeout) * time.Second)
	}

	return conn, nil
}

// Authenticate authenticates a user against LDAP and returns the user and their groups.
func (p *LDAPProvider) Authenticate(username, password string) (*models.User, []string, error) {
	conn, err := p.Connect()
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if errClose := conn.Close(); errClose != nil {
			log.Warn().Err(errClose).Msg("failed to close LDAP connection")
		}
	}()

	if errBindService := p.bindServiceForSearch(conn); errBindService != nil {
		return nil, nil, errBindService
	}

	userEntry, errSearch := p.searchUserEntry(conn, username)
	if errSearch != nil {
		return nil, nil, errSearch
	}

	userDN := userEntry.DN

	if errAuthAsUser := p.authenticateAsUser(conn, userDN, password); errAuthAsUser != nil {
		return nil, nil, errAuthAsUser
	}

	email := userEntry.GetAttributeValue(p.config.EmailAttr)
	firstName := userEntry.GetAttributeValue(p.config.FirstNameAttr)
	lastName := userEntry.GetAttributeValue(p.config.LastNameAttr)

	if errRebind := p.rebindServiceForGroups(conn); errRebind != nil {
		return nil, nil, errRebind
	}

	groups, errUserGroup := p.getUserGroups(conn, userDN)
	if errUserGroup != nil {
		return nil, nil, fmt.Errorf("failed to get user groups: %w", errUserGroup)
	}

	user, errUpsert := p.upsertLDAPUser(username, userDN, email, firstName, lastName)
	if errUpsert != nil {
		return nil, nil, errUpsert
	}

	return user, groups, nil
}

// bindServiceForSearch binds with the configured service account (if provided)
// to perform user search. Returns a wrapped error on failure.
func (p *LDAPProvider) bindServiceForSearch(conn *ldap.Conn) error {
	if p.config.BindDN == "" {
		return nil
	}

	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		return fmt.Errorf("failed to bind with service account: %w", err)
	}

	return nil
}

// rebindServiceForGroups re-binds with the service account (if provided)
// to perform group searches after authenticating as the user.
func (p *LDAPProvider) rebindServiceForGroups(conn *ldap.Conn) error {
	if p.config.BindDN == "" {
		return nil
	}

	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		return fmt.Errorf("failed to re-bind with service account: %w", err)
	}

	return nil
}

// searchUserEntry searches LDAP for the given username and returns a single entry.
func (p *LDAPProvider) searchUserEntry(conn *ldap.Conn, username string) (*ldap.Entry, error) {
	userFilter := strings.ReplaceAll(p.config.UserFilter, "{username}", ldap.EscapeFilter(username))
	searchRequest := ldap.NewSearchRequest(
		p.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // Size limit
		p.config.Timeout,
		false,
		userFilter,
		[]string{
			p.config.UsernameAttr,
			p.config.EmailAttr,
			p.config.FirstNameAttr,
			p.config.LastNameAttr,
			"dn",
		},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user: %w", err)
	}

	switch len(searchResult.Entries) {
	case 0:
		return nil, ErrUserNotFound
	case 1:
		return searchResult.Entries[0], nil
	default:
		return nil, ErrMultipleUsersFound
	}
}

// authenticateAsUser binds to LDAP using the user's DN and password.
func (p *LDAPProvider) authenticateAsUser(conn *ldap.Conn, userDN, password string) error {
	if err := conn.Bind(userDN, password); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

// upsertLDAPUser creates or updates a user record based on LDAP attributes.
func (p *LDAPProvider) upsertLDAPUser(username, userDN, email, firstName, lastName string) (*models.User, error) {
	var user models.User

	err := p.db.Where("external_id = ? AND auth_source = ?", userDN, models.AuthSourceLDAP).
		First(&user).Error

	notFound := errors.Is(err, gorm.ErrRecordNotFound)

	if notFound {
		user = models.User{
			Active:     true,
			Username:   username,
			Email:      email,
			FirstName:  firstName,
			LastName:   lastName,
			AuthSource: models.AuthSourceLDAP,
			ExternalID: userDN,
			RoleID:     0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err = p.db.Create(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		return &user, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Update existing user
	user.Email = email
	user.FirstName = firstName
	user.LastName = lastName
	user.UpdatedAt = time.Now()

	if err = p.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

// getUserGroups retrieves all groups a user belongs to from LDAP.
func (p *LDAPProvider) getUserGroups(conn *ldap.Conn, userDN string) ([]string, error) {
	if p.config.GroupBaseDN == "" {
		return nil, nil
	}

	groupFilter := strings.ReplaceAll(p.config.GroupFilter, "{userdn}", ldap.EscapeFilter(userDN))
	searchRequest := ldap.NewSearchRequest(
		p.config.GroupBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		p.config.Timeout,
		false,
		groupFilter,
		[]string{p.config.GroupNameAttr, "dn"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search for groups: %w", err)
	}

	groups := make([]string, len(searchResult.Entries))
	for i, entry := range searchResult.Entries {
		// Use DN as the group identifier for mapping
		groups[i] = entry.DN
	}

	return groups, nil
}

// SearchUsers searches for users in LDAP using a custom filter.
// This is useful for administrative purposes such as user lookup or synchronization.
// The filter should be a valid LDAP search filter, and limit restricts the number of results.
func (p *LDAPProvider) SearchUsers(filter string, limit int) ([]*ldap.Entry, error) {
	conn, err := p.Connect()
	if err != nil {
		return nil, err
	}

	defer func() {
		if errClose := conn.Close(); errClose != nil {
			log.Warn().Err(errClose).Msg("failed to close LDAP connection")
		}
	}()

	// Bind with service account
	if p.config.BindDN != "" {
		if err = conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return nil, fmt.Errorf("failed to bind: %w", err)
		}
	}

	searchRequest := ldap.NewSearchRequest(
		p.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		limit,
		p.config.Timeout,
		false,
		filter,
		[]string{
			p.config.UsernameAttr,
			p.config.EmailAttr,
			p.config.FirstNameAttr,
			p.config.LastNameAttr,
			"dn",
		},
		nil,
	)

	searchResult, errSearch := conn.Search(searchRequest)
	if errSearch != nil {
		return nil, fmt.Errorf("failed to search: %w", errSearch)
	}

	return searchResult.Entries, nil
}

// TestConnection tests the LDAP server connection and bind credentials.
// It establishes a connection and attempts to bind with the configured service account.
// Returns nil if the connection and bind are successful, otherwise returns an error.
func (p *LDAPProvider) TestConnection() error {
	conn, err := p.Connect()
	if err != nil {
		return err
	}

	defer func() {
		if errClose := conn.Close(); errClose != nil {
			log.Warn().Err(errClose).Msg("failed to close LDAP connection")
		}
	}()

	// Try to bind
	if p.config.BindDN != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return fmt.Errorf("bind failed: %w", err)
		}
	}

	return nil
}
