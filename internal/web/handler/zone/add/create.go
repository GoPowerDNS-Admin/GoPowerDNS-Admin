package zoneadd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

// resolveZoneName sets form.Name based on the zone type.
// For reverse zones it computes the name from the CIDR; for forward zones it
// ensures a trailing dot.
func resolveZoneName(form *ZoneForm) error {
	switch form.ZoneType {
	case ZoneTypeReverseIPv4:
		name, err := ReverseIPv4Zone(form.ReverseNetwork)
		if err != nil {
			return fmt.Errorf("invalid IPv4 network: %w", err)
		}

		form.Name = name

	case ZoneTypeReverseIPv6:
		name, err := ReverseIPv6Zone(form.ReverseNetwork)
		if err != nil {
			return fmt.Errorf("invalid IPv6 network: %w", err)
		}

		form.Name = name

	case ZoneTypeForward:
		if !strings.HasSuffix(form.Name, ".") {
			form.Name += "."
		}
	}

	return nil
}

// createZone creates the zone in PowerDNS according to form.Kind.
func createZone(ctx context.Context, form *ZoneForm) error {
	// Validate kind before making any API calls.
	switch form.Kind {
	case ZoneKindNative, ZoneKindMaster, ZoneKindSlave:
		// valid
	default:
		return fmt.Errorf("unknown zone kind: %s", form.Kind)
	}

	if powerdns.Engine.Client == nil {
		return errors.New("PowerDNS client is not initialized")
	}

	soaEditAPIStr := string(form.SOAEditAPI)

	switch form.Kind {
	case ZoneKindNative:
		_, err := powerdns.Engine.Zones.AddNative(
			ctx, form.Name,
			false, "", false, "", soaEditAPIStr, false, nil,
		)

		return err
	case ZoneKindMaster:
		_, err := powerdns.Engine.Zones.AddMaster(
			ctx, form.Name,
			false, "", false, "", soaEditAPIStr, false, nil,
		)

		return err
	case ZoneKindSlave:
		var masters []string

		if form.Masters != "" {
			for master := range strings.SplitSeq(form.Masters, ",") {
				masters = append(masters, strings.TrimSpace(master))
			}
		}

		_, err := powerdns.Engine.Zones.AddSlave(ctx, form.Name, masters)

		return err
	}

	return fmt.Errorf("unknown zone kind: %s", form.Kind)
}
