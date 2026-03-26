package daemon

import (
	"context"
	"time"

	pdnsapi "github.com/joeig/go-powerdns/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/powerdns"
)

const (
	demoTimeout = 30 * time.Second
	demoTTL300  = uint32(300)
	demoTTL3600 = uint32(3600)
)

// seedDemoUser creates a standard "user" role account for demo exploration.
// It is idempotent — a second call is a no-op if the account already exists.
func seedDemoUser(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Where("username = ?", "user").Count(&count)

	if count > 0 {
		return
	}

	var userRole models.Role
	db.Where(models.WhereNameIs, "user").First(&userRole)

	u := &models.User{
		Username:    "user",
		Email:       "user@demo.local",
		Password:    models.HashPassword("password"),
		Active:      true,
		RoleID:      userRole.ID,
		AuthSource:  models.AuthSourceLocal,
		DisplayName: "Demo User",
	}

	if err := db.Create(u).Error; err != nil {
		log.Error().Err(err).Msg("Failed to create demo user")
	} else {
		log.Info().Msg("Created demo user (username: user, password: password)")
	}
}

// seedDemoZones creates a small set of pre-populated DNS zones so visitors can
// explore the UI without having to configure anything. It is idempotent — zones
// that already exist are silently skipped.
func seedDemoZones() {
	if powerdns.Engine.Client == nil {
		log.Warn().Msg("PowerDNS client not available, skipping demo zone seeding")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), demoTimeout)
	defer cancel()

	// Build set of already-existing zone names.
	existing, err := powerdns.Engine.Zones.List(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list zones, skipping demo zone seeding")
		return
	}

	have := make(map[string]bool, len(existing))
	for i := range existing {
		if existing[i].Name != nil {
			have[*existing[i].Name] = true
		}
	}

	createExampleCom(ctx, have)
	createExampleOrg(ctx, have)
	createReverseZone(ctx, have)
}

// ptr returns a pointer to v — shorthand used when building pdnsapi structs.
func ptr[T any](v T) *T { return &v }

// createZoneNative creates a Native zone and patches the given RRsets into it.
// If the zone already exists the call is skipped entirely.
func createZoneNative(ctx context.Context, name string, have map[string]bool, rrsets []pdnsapi.RRset) {
	if have[name] {
		log.Debug().Str("zone", name).Msg("Demo zone already exists, skipping")
		return
	}

	if _, err := powerdns.Engine.Zones.AddNative(
		ctx, name, false, "", false, "", "DEFAULT", false, nil,
	); err != nil {
		log.Error().Err(err).Str("zone", name).Msg("Failed to create demo zone")
		return
	}

	if err := powerdns.Engine.Records.Patch(ctx, name, &pdnsapi.RRsets{Sets: rrsets}); err != nil {
		log.Error().Err(err).Str("zone", name).Msg("Failed to patch demo records")
		return
	}

	log.Info().Str("zone", name).Msg("Seeded demo zone")
}

func createExampleCom(ctx context.Context, have map[string]bool) {
	replace := pdnsapi.ChangeTypeReplace
	rrtA := pdnsapi.RRTypeA
	rrtAAAA := pdnsapi.RRTypeAAAA
	rrtMX := pdnsapi.RRTypeMX
	rrtTXT := pdnsapi.RRTypeTXT
	rrtCNAME := pdnsapi.RRTypeCNAME
	rrtCAA := pdnsapi.RRTypeCAA
	rrtSRV := pdnsapi.RRTypeSRV

	ttl300 := demoTTL300
	ttl3600 := demoTTL3600

	createZoneNative(ctx, "example.com.", have, []pdnsapi.RRset{
		{
			Name: ptr("example.com."), Type: &rrtA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("203.0.113.10"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("www.example.com."), Type: &rrtA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("203.0.113.10"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("mail.example.com."), Type: &rrtA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("203.0.113.20"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("example.com."), Type: &rrtAAAA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("2001:db8::10"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("www.example.com."), Type: &rrtAAAA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("2001:db8::10"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("example.com."), Type: &rrtMX, TTL: &ttl3600,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("10 mail.example.com."), Disabled: ptr(false)}},
		},
		{
			Name: ptr("example.com."), Type: &rrtTXT, TTL: &ttl3600,
			ChangeType: &replace,
			Records: []pdnsapi.Record{
				{Content: ptr(`"v=spf1 mx -all"`), Disabled: ptr(false)},
			},
		},
		{
			Name: ptr("_dmarc.example.com."), Type: &rrtTXT, TTL: &ttl3600,
			ChangeType: &replace,
			Records: []pdnsapi.Record{
				{Content: ptr(`"v=DMARC1; p=none; rua=mailto:dmarc@example.com"`), Disabled: ptr(false)},
			},
		},
		{
			Name: ptr("blog.example.com."), Type: &rrtCNAME, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("www.example.com."), Disabled: ptr(false)}},
		},
		{
			Name: ptr("ftp.example.com."), Type: &rrtCNAME, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("www.example.com."), Disabled: ptr(false)}},
		},
		{
			Name: ptr("example.com."), Type: &rrtCAA, TTL: &ttl3600,
			ChangeType: &replace,
			Records: []pdnsapi.Record{
				{Content: ptr(`0 issue "letsencrypt.org"`), Disabled: ptr(false)},
			},
		},
		{
			Name: ptr("_sip._tcp.example.com."), Type: &rrtSRV, TTL: &ttl3600,
			ChangeType: &replace,
			Records: []pdnsapi.Record{
				{Content: ptr("10 20 5060 sip.example.com."), Disabled: ptr(false)},
			},
		},
	})
}

func createExampleOrg(ctx context.Context, have map[string]bool) {
	replace := pdnsapi.ChangeTypeReplace
	rrtA := pdnsapi.RRTypeA
	rrtAAAA := pdnsapi.RRTypeAAAA
	rrtMX := pdnsapi.RRTypeMX
	rrtTXT := pdnsapi.RRTypeTXT
	rrtCNAME := pdnsapi.RRTypeCNAME

	ttl300 := demoTTL300
	ttl3600 := demoTTL3600

	createZoneNative(ctx, "example.org.", have, []pdnsapi.RRset{
		{
			Name: ptr("example.org."), Type: &rrtA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("198.51.100.1"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("www.example.org."), Type: &rrtA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("198.51.100.1"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("api.example.org."), Type: &rrtA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("198.51.100.2"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("example.org."), Type: &rrtAAAA, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("2001:db8:1::1"), Disabled: ptr(false)}},
		},
		{
			Name: ptr("example.org."), Type: &rrtMX, TTL: &ttl3600,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("10 mail.example.org."), Disabled: ptr(false)}},
		},
		{
			Name: ptr("example.org."), Type: &rrtTXT, TTL: &ttl3600,
			ChangeType: &replace,
			Records: []pdnsapi.Record{
				{Content: ptr(`"v=spf1 a mx -all"`), Disabled: ptr(false)},
			},
		},
		{
			Name: ptr("cdn.example.org."), Type: &rrtCNAME, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("www.example.org."), Disabled: ptr(false)}},
		},
	})
}

func createReverseZone(ctx context.Context, have map[string]bool) {
	replace := pdnsapi.ChangeTypeReplace
	rrtPTR := pdnsapi.RRTypePTR

	ttl300 := demoTTL300

	createZoneNative(ctx, "113.0.203.in-addr.arpa.", have, []pdnsapi.RRset{
		{
			Name: ptr("10.113.0.203.in-addr.arpa."), Type: &rrtPTR, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("example.com."), Disabled: ptr(false)}},
		},
		{
			Name: ptr("20.113.0.203.in-addr.arpa."), Type: &rrtPTR, TTL: &ttl300,
			ChangeType: &replace,
			Records:    []pdnsapi.Record{{Content: ptr("mail.example.com."), Disabled: ptr(false)}},
		},
	})
}
