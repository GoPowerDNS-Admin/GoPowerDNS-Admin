package zoneedit

import (
	"github.com/rs/zerolog/log"

	zonesettings "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/zone"
)

// loadAllowedRecordTypes returns the allowed record types based on settings, with safe defaults.
func (s *Service) loadAllowedRecordTypes() []RecordTypeOption {
	var (
		recordSettings     zonesettings.RecordSettings
		allowedRecordTypes []RecordTypeOption
	)

	if err := recordSettings.Load(s.db); err != nil {
		log.Warn().Err(err).Msg("failed to load zone record settings, using defaults")
		// Default record types if settings not found
		allowedRecordTypes = []RecordTypeOption{
			{
				Type:        "A",
				Description: "IPv4 Address",
				Help:        "Enter an IPv4 address, e.g., 203.0.113.5",
			},
			{
				Type:        "AAAA",
				Description: "IPv6 Address",
				Help:        "Enter an IPv6 address, e.g., 2001:db8::1",
			},
			{
				Type:        "CNAME",
				Description: "Canonical Name (Alias)",
				Help:        "Target hostname (FQDN). A trailing dot is added automatically.",
			},
			{
				Type:        "MX",
				Description: "Mail Exchange",
				Help:        "Format: priority hostname (e.g., 10 mail.example.com). Hostname canonicalized.",
			},
			{
				Type:        "NS",
				Description: "Name Server",
				Help:        "Nameserver hostname (FQDN). Trailing dot added automatically.",
			},
			{
				Type:        "PTR",
				Description: "Pointer (Reverse DNS)",
				Help:        "Target hostname (FQDN). Trailing dot added automatically.",
			},
			{
				Type:        "SOA",
				Description: "Start of Authority",
				Help:        "Edit SOA via the SOA editor (click Edit on existing SOA).",
			},
			{
				Type:        "SRV",
				Description: "Service Locator",
				Help:        "Format: priority weight port target. Target is canonicalized.",
			},
			{
				Type:        "TXT",
				Description: "Text Record",
				Help:        "Plain text. Quotes will be added automatically if missing.",
			},
			{
				Type:        "CAA",
				Description: "Certification Authority Authorization",
				Help:        `Format: flags tag "value" (e.g., 0 issue "letsencrypt.org").`,
			},
		}

		return allowedRecordTypes
	}

	for recordType, settings := range recordSettings.Records {
		if settings.Forward {
			allowedRecordTypes = append(allowedRecordTypes, RecordTypeOption{
				Type:        recordType,
				Description: settings.Description,
				Help:        settings.Help,
			})
		}
	}

	return allowedRecordTypes
}
