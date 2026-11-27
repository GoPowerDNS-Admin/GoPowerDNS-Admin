package zoneedit

import (
	"github.com/rs/zerolog/log"

	zonesettings "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/zone"
)

// loadAllowedRecordTypes returns the allowed record types based on settings, with safe defaults.
func (s *Service) loadAllowedRecordTypes(reverse bool) []RecordTypeOption {
	var (
		recordSettings     zonesettings.RecordSettings
		allowedRecordTypes []RecordTypeOption
		rto                RecordTypeOption
	)

	if err := recordSettings.Load(s.db); err != nil {
		log.Warn().Err(err).Msg("failed to load zone record settings")
		return allowedRecordTypes
	}

	for recordType, settings := range recordSettings.Records {
		if settings.Forward && !reverse || settings.Reverse && reverse {
			rto = RecordTypeOption{
				Type:        recordType,
				Description: settings.Description,
				Help:        settings.Help,
			}

			allowedRecordTypes = append(allowedRecordTypes, rto)
		}
	}

	return allowedRecordTypes
}

// disallowedTypeError reports a change to a record type that is not enabled
// in the zone record settings.
type disallowedTypeError struct {
	recordType string
}

func (e *disallowedTypeError) Error() string {
	return "Modification of record type " + e.recordType + " is not allowed"
}

// validateRecordsUpdateAreValidTypes checks if all provided record types are
// allowed. It returns a *disallowedTypeError for the first type that is not.
func (s *Service) validateRecordsUpdateAreValidTypes(
	zoneName string,
	request *RecordsUpdateRequest,
	reverse bool) error {
	allowedTypes := s.loadAllowedRecordTypes(reverse)

	allowedTypesMap := make(map[string]bool, len(allowedTypes))
	for _, at := range allowedTypes {
		allowedTypesMap[at.Type] = true
	}

	for _, change := range request.Changes {
		if !allowedTypesMap[change.Type] {
			// Allow editing records that already exist even if their type is not in
			// the allowed-types list (e.g. SOA records managed outside the admin UI).
			if change.Existed {
				continue
			}

			log.Warn().Str("zone_name", zoneName).Str("record_type", change.Type).
				Msg("attempt to modify disallowed record type")

			return &disallowedTypeError{recordType: change.Type}
		}
	}

	return nil
}
