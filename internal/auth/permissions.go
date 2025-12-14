package auth

// Permission constants define the available permissions in the system.
// These are used for role-based access control (RBAC) to restrict access
// to specific resources and actions.
const (
	// PermDashboardView allows viewing the main dashboard with DNS zones.
	PermDashboardView = "dashboard.view"

	// PermZoneCreate allows creating new DNS zones.
	PermZoneCreate = "zone.create"
	// PermZoneRead allows viewing DNS zone details and records.
	PermZoneRead = "zone.read"
	// PermZoneUpdate allows editing DNS zone settings and records.
	PermZoneUpdate = "zone.update"
	// PermZoneDelete allows deleting DNS zones.
	PermZoneDelete = "zone.delete"
	// PermZoneList allows listing all DNS zones.
	PermZoneList = "zone.list"

	// PermAdminSettings allows managing application-wide settings.
	PermAdminSettings = "admin.settings"
	// PermAdminServerConfig allows viewing PowerDNS server configuration.
	PermAdminServerConfig = "admin.server.config"
	// PermAdminPDNSServer allows managing PowerDNS server connection settings.
	PermAdminPDNSServer = "admin.pdns.server"
	// PermAdminZoneRecords allows managing DNS record type permissions.
	PermAdminZoneRecords = "admin.zone.records"
	// PermAdminUsers allows managing user accounts.
	PermAdminUsers = "admin.users"
	// PermAdminRoles allows managing roles and their permissions.
	PermAdminRoles = "admin.roles"
	// PermAdminGroups allows managing user groups.
	PermAdminGroups = "admin.groups"
	// PermAdminGroupMappings allows managing mappings between external groups and internal roles.
	PermAdminGroupMappings = "admin.group.mappings"
)
