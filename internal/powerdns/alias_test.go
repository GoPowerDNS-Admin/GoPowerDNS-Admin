package powerdns

import (
	"errors"
	"testing"

	pdnsapi "github.com/joeig/go-powerdns/v3"
)

func setting(name, value string) pdnsapi.ConfigSetting {
	return pdnsapi.ConfigSetting{Name: &name, Value: &value}
}

func TestAliasSupportFromSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings []pdnsapi.ConfigSetting
		want     AliasSupport
		enabled  bool
	}{
		{
			name:     "defaults",
			settings: []pdnsapi.ConfigSetting{setting("expand-alias", "no"), setting("resolver", "")},
			want:     AliasSupport{},
		},
		{
			name:     "expand-alias without resolver",
			settings: []pdnsapi.ConfigSetting{setting("expand-alias", "yes"), setting("resolver", "")},
			want:     AliasSupport{ExpandAlias: true},
		},
		{
			name:     "resolver without expand-alias",
			settings: []pdnsapi.ConfigSetting{setting("expand-alias", "no"), setting("resolver", "10.10.0.1")},
			want:     AliasSupport{Resolver: "10.10.0.1"},
		},
		{
			name:     "fully configured",
			settings: []pdnsapi.ConfigSetting{setting("expand-alias", "yes"), setting("resolver", " 10.10.0.1 ")},
			want:     AliasSupport{ExpandAlias: true, Resolver: "10.10.0.1"},
			enabled:  true,
		},
		{
			name:     "settings missing",
			settings: []pdnsapi.ConfigSetting{setting("loglevel", "5"), {Name: nil}},
			want:     AliasSupport{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AliasSupportFromSettings(tc.settings)
			if got != tc.want {
				t.Fatalf("want %+v, got %+v", tc.want, got)
			}

			if got.Enabled() != tc.enabled {
				t.Fatalf("Enabled(): want %v, got %v", tc.enabled, got.Enabled())
			}

			if (got.Problem() == "") != tc.enabled {
				t.Fatalf("Problem() = %q does not match Enabled() = %v", got.Problem(), tc.enabled)
			}
		})
	}
}

func TestAliasSupportProblemNamesTheMissingSetting(t *testing.T) {
	tests := []struct {
		support AliasSupport
		want    string
	}{
		{AliasSupport{}, "ALIAS records are not resolved by the PowerDNS server: set expand-alias=yes and configure a resolver"},
		{AliasSupport{Resolver: "10.10.0.1"}, "ALIAS records are not resolved by the PowerDNS server: expand-alias is disabled"},
		{AliasSupport{ExpandAlias: true}, "ALIAS records are not resolved by the PowerDNS server: no resolver is configured"},
	}

	for _, tc := range tests {
		if got := tc.support.Problem(); got != tc.want {
			t.Errorf("Problem() for %+v: want %q, got %q", tc.support, tc.want, got)
		}
	}
}

func TestCheckAliasSupportWithoutClient(t *testing.T) {
	if _, err := (engine{}).CheckAliasSupport(t.Context()); !errors.Is(err, ErrClientNotInitialized) {
		t.Fatalf("want ErrClientNotInitialized, got %v", err)
	}
}
