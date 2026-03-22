package daemon

import (
	"errors"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/config"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/controller/setting"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	ttlsettings "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/ttl"
	zonesettings "github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler/admin/settings/zone"
)

func seed(cfg *config.Config, db *gorm.DB) {
	// Seed roles
	seedRoles(db)

	// Seed permissions
	seedPermissions(db)

	// Seed role-permission mappings
	seedRolePermissions(db)

	// Seed default admin user
	seedUsers(db)

	// Seed zone record settings from config (only if not already set)
	seedZoneRecordSettings(cfg, db)

	// Seed TTL presets (only if not already set)
	seedTTLPresets(db)
}

// seedRoles creates default roles.
func seedRoles(db *gorm.DB) {
	roles := []models.Role{
		{
			Name:        "admin",
			Description: "Full system access with all permissions",
			IsSystem:    true,
		},
		{
			Name:        "user",
			Description: "Standard user with zone management permissions",
			IsSystem:    true,
		},
		{
			Name:        "viewer",
			Description: "Read-only access to zones and dashboards",
			IsSystem:    true,
		},
	}

	for _, role := range roles {
		var existingRole models.Role

		err := db.Where(models.WhereNameIs, role.Name).First(&existingRole).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = db.Create(&role).Error; err != nil {
				log.Error().Err(err).Str("role", role.Name).Msg("Failed to create role")
			} else {
				log.Info().Str("role", role.Name).Msg("Created role")
			}
		}
	}
}

// seedPermissions creates default permissions.
func seedPermissions(db *gorm.DB) {
	permissions := []models.Permission{
		// Dashboard permissions
		{
			Name:        "dashboard.view",
			Resource:    "dashboard",
			Action:      "view",
			Description: "View dashboard",
		},

		// Zone permissions
		{
			Name:        "zone.create",
			Resource:    "zone",
			Action:      "create",
			Description: "Create DNS zones",
		},
		{
			Name:        "zone.read",
			Resource:    "zone",
			Action:      "read",
			Description: "View DNS zones",
		},
		{
			Name:        "zone.update",
			Resource:    "zone",
			Action:      "update",
			Description: "Update DNS zones",
		},
		{
			Name:        "zone.delete",
			Resource:    "zone",
			Action:      "delete",
			Description: "Delete DNS zones",
		},
		{
			Name:        "zone.list",
			Resource:    "zone",
			Action:      "list",
			Description: "List DNS zones",
		},

		// Admin permissions
		{
			Name:        "admin.settings",
			Resource:    "admin",
			Action:      "settings",
			Description: "Manage application settings",
		},
		{
			Name:        "admin.server.config",
			Resource:    "admin",
			Action:      "server.config",
			Description: "View server configuration",
		},
		{
			Name:        "admin.pdns.server",
			Resource:    "admin",
			Action:      "pdns.server",
			Description: "Manage PowerDNS server settings",
		},
		{
			Name:        "admin.zone.records",
			Resource:    "admin",
			Action:      "zone.records",
			Description: "Manage zone record type settings",
		},
		{
			Name:        "admin.users",
			Resource:    "admin",
			Action:      "users",
			Description: "Manage users",
		},
		{
			Name:        "admin.roles",
			Resource:    "admin",
			Action:      "roles",
			Description: "Manage roles",
		},
		{
			Name:        "admin.groups",
			Resource:    "admin",
			Action:      "groups",
			Description: "Manage groups",
		},
		{
			Name:        "admin.group.mappings",
			Resource:    "admin",
			Action:      "group.mappings",
			Description: "Manage group-to-role mappings",
		},
		{
			Name:        "admin.activity.log",
			Resource:    "admin",
			Action:      "activity.log",
			Description: "View the activity / audit log",
		},
		{
			Name:        "admin.activity.log.undo",
			Resource:    "admin",
			Action:      "activity.log.undo",
			Description: "Undo record changes from the activity log",
		},
		{
			Name:        "admin.tags",
			Resource:    "admin",
			Action:      "tags",
			Description: "Manage zone-access tags",
		},
		{
			Name:        "admin.zone.tags",
			Resource:    "admin",
			Action:      "zone.tags",
			Description: "Assign tags to zones",
		},
		{
			Name:        "admin.ttl.presets",
			Resource:    "admin",
			Action:      "ttl.presets",
			Description: "Manage global TTL preset values",
		},
	}

	for _, perm := range permissions {
		var existingPerm models.Permission

		err := db.Where(models.WhereNameIs, perm.Name).First(&existingPerm).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = db.Create(&perm).Error; err != nil {
				log.Error().Err(err).Str("permission", perm.Name).Msg("Failed to create permission")
			} else {
				log.Debug().Str("permission", perm.Name).Msg("Created permission")
			}
		}
	}
}

// seedRolePermissions creates role-permission mappings.
func seedRolePermissions(db *gorm.DB) {
	// Get roles
	var adminRole, userRole, viewerRole models.Role
	db.Where(models.WhereNameIs, "admin").First(&adminRole)
	db.Where(models.WhereNameIs, "user").First(&userRole)
	db.Where(models.WhereNameIs, "viewer").First(&viewerRole)

	// Get all permissions
	var allPermissions []models.Permission
	db.Find(&allPermissions)

	// Admin gets all permissions
	for _, perm := range allPermissions {
		var existing models.RolePermission

		err := db.Where("role_id = ? AND permission_id = ?", adminRole.ID, perm.ID).
			First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			db.Create(&models.RolePermission{
				RoleID:       adminRole.ID,
				PermissionID: perm.ID,
			})
		}
	}

	// User gets zone and dashboard permissions
	userPermissions := []string{
		"dashboard.view",
		"zone.create",
		"zone.read",
		"zone.update",
		"zone.delete",
		"zone.list",
		"admin.activity.log",
	}
	assignPermissionsToRole(db, userRole.ID, userPermissions)

	// Viewer gets read-only permissions
	viewerPermissions := []string{
		"dashboard.view",
		"zone.read",
		"zone.list",
		"admin.server.config",
		"admin.activity.log",
	}
	assignPermissionsToRole(db, viewerRole.ID, viewerPermissions)

	log.Info().Msg("Role-permission mappings created")
}

// assignPermissionsToRole assigns a list of permission names to a role.
func assignPermissionsToRole(db *gorm.DB, roleID uint, permissionNames []string) {
	for _, permName := range permissionNames {
		var perm models.Permission
		if err := db.Where(models.WhereNameIs, permName).First(&perm).Error; err == nil {
			var existing models.RolePermission

			err := db.Where("role_id = ? AND permission_id = ?", roleID, perm.ID).
				First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				db.Create(&models.RolePermission{
					RoleID:       roleID,
					PermissionID: perm.ID,
				})
			}
		}
	}
}

// defaultRecordSettings defines the built-in DNS record type defaults.
// These can be overridden or extended via the [record] section in main.toml.
//

var defaultRecordSettings = config.Record{
	"A": {
		Forward: true, Reverse: false,
		Description: "IPv4 Address",
		Help:        "Enter an IPv4 address (e.g., 203.0.113.5).",
	},
	"AAAA": {
		Forward: true, Reverse: false,
		Description: "IPv6 Address",
		Help:        "Enter an IPv6 address (e.g., 2001:db8::1).",
	},
	"AFSDB": {
		Forward: false, Reverse: false,
		Description: "AFS Database Location",
		Help:        "AFS database location; advanced record.",
	},
	"ALIAS": {
		Forward: false, Reverse: false,
		Description: "Auto-resolved Alias",
		Help:        "Alias to another hostname (provider-specific).",
	},
	"CAA": {
		Forward: true, Reverse: false,
		Description: "Certification Authority Authorization",
		Help:        `Format: flags tag "value" (e.g., 0 issue "letsencrypt.org").`,
	},
	"CERT": {
		Forward: false, Reverse: false,
		Description: "Certificate Record",
		Help:        "Certificate record; provide type, key tag, algorithm, and certificate data.",
	},
	"CDNSKEY": {
		Forward: false, Reverse: false,
		Description: "Child DNSKEY",
		Help:        "Child zone DNSKEY (DNSSEC).",
	},
	"CDS": {
		Forward: false, Reverse: false,
		Description: "Child Delegation Signer",
		Help:        "Child DS record (DNSSEC).",
	},
	"CNAME": {
		Forward: true, Reverse: false,
		Description: "Canonical Name (Alias)",
		Help:        "Enter the target hostname (FQDN). A trailing dot will be added automatically.",
	},
	"DNSKEY": {
		Forward: false, Reverse: false,
		Description: "DNS Public Key",
		Help:        "DNSKEY record (DNSSEC).",
	},
	"DNAME": {
		Forward: false, Reverse: false,
		Description: "Delegation Name",
		Help:        "DNAME redirection of a subtree to another domain.",
	},
	"DS": {
		Forward: false, Reverse: false,
		Description: "Delegation Signer",
		Help:        "DS record (DNSSEC).",
	},
	"DLV": {
		Forward: false, Reverse: false,
		Description: "DNSSEC Lookaside Validation",
		Help:        "DLV (deprecated).",
	},
	"HINFO": {
		Forward: false, Reverse: false,
		Description: "Host Information",
		Help:        `Host hardware and OS (e.g., "Intel-386" "Unix").`,
	},
	"KEY": {
		Forward: false, Reverse: false,
		Description: "Key Record",
		Help:        "KEY record (obsolete; use DNSKEY).",
	},
	"LOC": {
		Forward: true, Reverse: true,
		Description: "Location Information",
		Help:        "Geographical location (e.g., 52 22 23.000 N 4 53 32.000 E 0.00m).",
	},
	"LUA": {
		Forward: false, Reverse: false,
		Description: "LUA Record",
		Help:        "Lua record (PowerDNS-specific).",
	},
	"MX": {
		Forward: true, Reverse: false,
		Description: "Mail Exchange",
		Help:        "Format: priority hostname (e.g., 10 mail.example.com). Hostname will be canonicalized.",
	},
	"NAPTR": {
		Forward: false, Reverse: false,
		Description: "Naming Authority Pointer",
		Help:        "Complex structured record: order preference flags service regexp replacement.",
	},
	"NS": {
		Forward: true, Reverse: true,
		Description: "Name Server",
		Help:        "Enter the nameserver hostname (FQDN). A trailing dot will be added automatically.",
	},
	"OPENPGPKEY": {
		Forward: false, Reverse: false,
		Description: "OpenPGP Public Key",
		Help:        "OpenPGP public key record.",
	},
	"PTR": {
		Forward: true, Reverse: true,
		Description: "Pointer (Reverse DNS)",
		Help: "Enter the target hostname (FQDN) for reverse DNS." +
			" A trailing dot will be added automatically.",
	},
	"RP": {
		Forward: false, Reverse: false,
		Description: "Responsible Person",
		Help:        "Mailbox and TXT pointer (e.g., hostmaster.example.com. txt-host.example.com.).",
	},
	"SOA": {
		Forward: false, Reverse: false,
		Description: "Start of Authority",
		Help:        "Use the SOA editor by editing an existing SOA record.",
	},
	"SPF": {
		Forward: true, Reverse: false,
		Description: "Sender Policy Framework",
		Help:        "SPF policy text. Quotes will be added automatically if missing.",
	},
	"SSHFP": {
		Forward: false, Reverse: false,
		Description: "SSH Fingerprint",
		Help:        "Format: algorithm fingerprint-type fingerprint (hex).",
	},
	"SRV": {
		Forward: true, Reverse: false,
		Description: "Service Locator",
		Help:        "Format: priority weight port target. Target will be canonicalized.",
	},
	"TKEY": {
		Forward: false, Reverse: false,
		Description: "Transaction Key",
		Help:        "TKEY (transaction key).",
	},
	"TSIG": {
		Forward: false, Reverse: false,
		Description: "Transaction Signature",
		Help:        "TSIG shared-secret signature.",
	},
	"TLSA": {
		Forward: false, Reverse: false,
		Description: "TLS Authentication",
		Help:        "Format: usage selector matching-type certificate-association-data.",
	},
	"SMIMEA": {
		Forward: false, Reverse: false,
		Description: "S/MIME Certificate Association",
		Help:        "Format: usage selector matching-type certificate-association-data.",
	},
	"TXT": {
		Forward: true, Reverse: true,
		Description: "Text Record",
		Help:        "Text value. Quotes are not required; they will be added automatically if missing.",
	},
	"URI": {
		Forward: false, Reverse: false,
		Description: "Uniform Resource Identifier",
		Help: `Format: priority weight "https://...".` +
			" If only a URL is provided, 0 0 will be defaulted and URL quoted.",
	},
}

// seedZoneRecordSettings ensures the zone record type settings in the database
// are up to date on every startup:
//
//   - First run (no DB entry or empty): seed built-in defaults merged with TOML.
//   - Subsequent runs: add any TOML entries not yet in the DB without touching
//     existing records (preserving admin-UI changes).
func seedZoneRecordSettings(cfg *config.Config, db *gorm.DB) {
	existing, err := setting.Get(db, zonesettings.SettingKeyZoneRecords)
	if err != nil && !errors.Is(err, setting.ErrSettingNotFound) {
		log.Error().Err(err).Msg("failed to check zone record settings")
		return
	}

	if err == nil {
		if syncExistingZoneRecordSettings(cfg, db, existing) {
			return
		}
	}

	seedDefaultZoneRecordSettings(cfg, db)
}

// syncExistingZoneRecordSettings updates an existing zone record settings entry
// with any new TOML entries and removes stale custom entries. Returns true when
// processing is complete (no further action needed), false when the caller
// should fall through to a fresh seed (entry was empty and has been deleted).
func syncExistingZoneRecordSettings(cfg *config.Config, db *gorm.DB, existing *models.Setting) bool {
	var rs zonesettings.RecordSettings
	if loadErr := rs.Load(db); loadErr != nil || len(rs.Records) == 0 {
		// Exists but empty — delete so seedDefaultZoneRecordSettings can recreate it.
		if delErr := setting.Delete(db, existing.ID); delErr != nil {
			log.Error().Err(delErr).Msg("failed to clear empty zone record settings")
		}

		return false
	}

	added := syncNewTOMLEntries(cfg, &rs)
	removed := removeStaleEntries(cfg, &rs)

	if added == 0 && removed == 0 {
		return true // nothing changed
	}

	if saveErr := rs.Save(db); saveErr != nil {
		log.Error().Err(saveErr).Msg("failed to sync zone record settings with TOML")
		return true
	}

	log.Info().Int("added", added).Int("removed", removed).
		Msg("synced zone record settings with TOML config")

	return true
}

// syncNewTOMLEntries adds TOML-defined record types not yet present in the DB settings.
func syncNewTOMLEntries(cfg *config.Config, rs *zonesettings.RecordSettings) int {
	added := 0

	for k, v := range cfg.Record {
		if _, exists := rs.Records[k]; !exists {
			rs.Records[k] = v
			added++
		}
	}

	return added
}

// removeStaleEntries removes entries that are neither a built-in default nor
// present in the current TOML config (i.e. custom entries that were deleted).
func removeStaleEntries(cfg *config.Config, rs *zonesettings.RecordSettings) int {
	removed := 0

	for k := range rs.Records {
		_, isBuiltin := defaultRecordSettings[k]
		_, isToml := cfg.Record[k]

		if !isBuiltin && !isToml {
			delete(rs.Records, k)

			removed++
		}
	}

	return removed
}

// seedDefaultZoneRecordSettings creates zone record settings from built-in
// defaults merged with any TOML overrides.
func seedDefaultZoneRecordSettings(cfg *config.Config, db *gorm.DB) {
	merged := make(config.Record, len(defaultRecordSettings))

	for k, v := range defaultRecordSettings {
		merged[k] = v
	}

	for k, v := range cfg.Record {
		merged[k] = v
	}

	rs := &zonesettings.RecordSettings{Records: merged}
	if err := rs.Save(db); err != nil {
		log.Error().Err(err).Msg("failed to seed zone record settings")
		return
	}

	log.Info().Int("record_types", len(merged)).Msg("seeded zone record settings")
}

// seedTTLPresets seeds default TTL presets if none are stored yet.
func seedTTLPresets(db *gorm.DB) {
	var s ttlsettings.Settings
	if err := s.Load(db); err == nil {
		return // already seeded
	}

	s.Presets = ttlsettings.DefaultPresets()
	if err := s.Save(db); err != nil {
		log.Error().Err(err).Msg("failed to seed TTL presets")
		return
	}

	log.Info().Int("presets", len(s.Presets)).Msg("seeded TTL presets")
}

// seedUsers creates the default admin user.
func seedUsers(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)

	if count == 0 {
		// Get admin role
		var adminRole models.Role
		db.Where(models.WhereNameIs, "admin").First(&adminRole)

		// Create default admin user
		user := &models.User{
			Username:    "admin",
			Email:       "admin@localhost",
			Password:    models.HashPassword("changeme"),
			Active:      true,
			RoleID:      adminRole.ID,
			AuthSource:  models.AuthSourceLocal,
			DisplayName: "System Administrator",
		}

		if err := db.Create(user).Error; err != nil {
			log.Error().Err(err).Msg("Failed to create default admin user")
		} else {
			log.Info().Msg("Created default admin user (username: admin, password: changeme)")
			log.Warn().Msg("SECURITY WARNING: Please change the default admin password immediately!")
		}
	}
}
