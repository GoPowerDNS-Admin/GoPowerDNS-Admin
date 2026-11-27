package zoneedit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

const aliasRecordType = "ALIAS"

// errAliasUnverified is returned when the PowerDNS configuration cannot be
// read, so ALIAS support is unknown.
var errAliasUnverified = errors.New("could not verify ALIAS support on the PowerDNS server")

// aliasConflictTypes lists record types that must not share a name with an
// ALIAS record. PowerDNS builds the A/AAAA answers from the ALIAS target, and
// a CNAME may not coexist with any other type.
var aliasConflictTypes = []string{"A", "AAAA", "CNAME"}

// lookupAliasSupport is replaced in tests.
var lookupAliasSupport = func(ctx context.Context) (powerdns.AliasSupport, error) {
	return powerdns.Engine.CheckAliasSupport(ctx)
}

// checkAliasChanges verifies that ALIAS records added by the changes can be
// served. It returns errAliasUnverified (wrapped) when the server
// configuration cannot be read, and a plain error for any other problem.
func checkAliasChanges(ctx context.Context, current []pdnsapi.RRset, changes []RecordChange) error {
	if !changesAddAlias(changes) {
		return nil
	}

	if conflicts := aliasConflicts(current, changes); len(conflicts) > 0 {
		return fmt.Errorf("ALIAS records cannot share a name with A, AAAA or CNAME records: %s",
			strings.Join(conflicts, ", "))
	}

	support, err := lookupAliasSupport(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", errAliasUnverified, err)
	}

	if problem := support.Problem(); problem != "" {
		return errors.New(problem)
	}

	return nil
}

// changesAddAlias reports whether the changes create or update ALIAS records.
// Deleting ALIAS records needs no server support.
func changesAddAlias(changes []RecordChange) bool {
	for _, change := range changes {
		if change.Type == aliasRecordType && change.Changed && len(change.Records) > 0 {
			return true
		}
	}

	return false
}

// aliasConflicts returns the names that would hold an ALIAS record next to an
// A, AAAA or CNAME record once the changes are applied. Each entry has the form
// "name (TYPE)".
func aliasConflicts(current []pdnsapi.RRset, changes []RecordChange) []string {
	present := make(map[string]map[string]bool)

	mark := func(name, rrType string, exists bool) {
		name = canonicalName(name)
		if present[name] == nil {
			present[name] = make(map[string]bool)
		}

		present[name][rrType] = exists
	}

	for _, rrSet := range current {
		if rrSet.Name == nil || rrSet.Type == nil {
			continue
		}

		mark(*rrSet.Name, string(*rrSet.Type), len(rrSet.Records) > 0)
	}

	for _, change := range changes {
		isDeletion := change.Existed && len(change.Records) == 0
		if !change.Changed && !isDeletion {
			continue
		}

		mark(change.Name, change.Type, len(change.Records) > 0)
	}

	var conflicts []string

	for name, types := range present {
		if !types[aliasRecordType] {
			continue
		}

		for _, conflictType := range aliasConflictTypes {
			if types[conflictType] {
				conflicts = append(conflicts, name+" ("+conflictType+")")
			}
		}
	}

	sort.Strings(conflicts)

	return conflicts
}

// annotateAliasHelp prefixes the ALIAS help text with a warning when the
// server does not resolve ALIAS records. The server configuration is only
// read when ALIAS is one of the offered types.
func annotateAliasHelp(ctx context.Context, types []RecordTypeOption) []RecordTypeOption {
	idx := -1

	for i := range types {
		if types[i].Type == aliasRecordType {
			idx = i
			break
		}
	}

	if idx < 0 {
		return types
	}

	support, err := lookupAliasSupport(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("could not read PowerDNS configuration for ALIAS help text")
		return types
	}

	if problem := support.Problem(); problem != "" {
		types[idx].Help = "Warning: " + problem + ". " + types[idx].Help
	}

	return types
}

func canonicalName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	return name
}
