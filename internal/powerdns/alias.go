package powerdns

import (
	"context"
	"strings"

	pdnsapi "github.com/joeig/go-powerdns/v3"
)

const (
	configExpandAlias = "expand-alias"
	configResolver    = "resolver"
)

// AliasSupport describes whether a PowerDNS server resolves ALIAS records.
// PowerDNS only answers A/AAAA queries for an ALIAS name when expand-alias is
// enabled and a resolver is configured. Without both, the name returns no
// address at all.
type AliasSupport struct {
	ExpandAlias bool
	Resolver    string
}

// Enabled reports whether the server resolves ALIAS records.
func (a AliasSupport) Enabled() bool {
	return a.ExpandAlias && a.Resolver != ""
}

// Problem returns a user-facing explanation of why ALIAS records are not
// resolved, or an empty string when they are.
func (a AliasSupport) Problem() string {
	const prefix = "ALIAS records are not resolved by the PowerDNS server: "

	switch {
	case a.Enabled():
		return ""
	case !a.ExpandAlias && a.Resolver == "":
		return prefix + "set expand-alias=yes and configure a resolver"
	case !a.ExpandAlias:
		return prefix + "expand-alias is disabled"
	default:
		return prefix + "no resolver is configured"
	}
}

// AliasSupportFromSettings derives the ALIAS support from a PowerDNS
// configuration listing.
func AliasSupportFromSettings(settings []pdnsapi.ConfigSetting) AliasSupport {
	var support AliasSupport

	for _, setting := range settings {
		if setting.Name == nil || setting.Value == nil {
			continue
		}

		value := strings.TrimSpace(*setting.Value)

		switch *setting.Name {
		case configExpandAlias:
			support.ExpandAlias = isEnabledValue(value)
		case configResolver:
			support.Resolver = value
		}
	}

	return support
}

// CheckAliasSupport reads the server configuration through the API and
// reports whether ALIAS records are resolved.
func (e engine) CheckAliasSupport(ctx context.Context) (AliasSupport, error) {
	if e.Client == nil {
		return AliasSupport{}, ErrClientNotInitialized
	}

	settings, err := e.Config.List(ctx)
	if err != nil {
		return AliasSupport{}, err
	}

	return AliasSupportFromSettings(settings), nil
}

func isEnabledValue(value string) bool {
	switch strings.ToLower(value) {
	case "yes", "true", "on", "1":
		return true
	default:
		return false
	}
}
