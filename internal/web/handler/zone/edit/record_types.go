package zoneedit

import (
	"github.com/gofiber/fiber/v2"
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

// validateRecordsUpdateAreValidTypes checks if all provided record types are allowed.
func (s *Service) validateRecordsUpdateAreValidTypes(
	c *fiber.Ctx,
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
			log.Warn().Str("zone_name", zoneName).Str("record_type", change.Type).
				Msg("attempt to modify disallowed record type")

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Modification of record type " + change.Type + " is not allowed",
			})
		}
	}

	return nil
}
