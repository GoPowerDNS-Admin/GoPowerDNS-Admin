package group

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/onsi/gomega"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/db/models"
	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/web/handler"
)

// When a step inside the update transaction fails, Update must stop right
// there: roll back once and show the error of the failed step. Before, the
// helpers rendered an error page but returned nil, so Update went on with the
// rolled-back transaction and the last render replaced the real message.
func TestUpdate_MappingFailure_StopsAndShowsTheFailedStep(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	user := createUser(t, db, "alice", role.ID)
	grp := createGroup(t, db, "team", models.GroupSourceLocal)
	addUserToGroup(t, db, user.ID, grp.ID)

	if err := db.Migrator().DropTable(&models.GroupMapping{}); err != nil {
		t.Fatalf("drop group mapping table: %v", err)
	}

	app, views := newCaptureApp(t, db)

	form := url.Values{
		"name":     {"renamed-team"},
		"source":   {"local"},
		"role_id":  {uintStr(uint64(role.ID))},
		"user_ids": {uintStr(user.ID)},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusInternalServerError))

	errData, ok := views.data()["Error"].(handler.ErrorData)
	g.Expect(ok).To(gomega.BeTrue(), "error page data missing")
	g.Expect(errData.Title).To(gomega.Equal("Save Failed"))
	g.Expect(errData.Message).To(gomega.Equal("Failed to update group role"))

	var stored models.Group
	g.Expect(db.First(&stored, grp.ID).Error).To(gomega.Succeed())
	g.Expect(stored.Name).To(gomega.Equal("team"), "group change must be rolled back")
	g.Expect(memberIDs(t, db, grp.ID)).To(gomega.ConsistOf(user.ID))
}

func TestUpdate_MembershipFailure_ShowsTheFailedStep(t *testing.T) {
	g := gomega.NewWithT(t)
	db := newTestDB(t)

	role := createRole(t, db, "viewer")
	user := createUser(t, db, "alice", role.ID)
	grp := createGroup(t, db, "team", models.GroupSourceLocal)

	if err := db.Migrator().DropTable(&models.UserGroup{}); err != nil {
		t.Fatalf("drop user group table: %v", err)
	}

	app, views := newCaptureApp(t, db)

	form := url.Values{
		"name":     {"renamed-team"},
		"source":   {"local"},
		"user_ids": {uintStr(user.ID)},
	}

	resp := doPost(t, app, fmt.Sprintf("%s/%d", Path, grp.ID), form)

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusInternalServerError))

	errData, ok := views.data()["Error"].(handler.ErrorData)
	g.Expect(ok).To(gomega.BeTrue(), "error page data missing")
	g.Expect(errData.Message).To(gomega.Equal("Failed to update group members"))

	var stored models.Group
	g.Expect(db.First(&stored, grp.ID).Error).To(gomega.Succeed())
	g.Expect(stored.Name).To(gomega.Equal("team"), "group change must be rolled back")
}
