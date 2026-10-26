package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/identity"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/app"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/password"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type fixture struct {
	store   *postgres.Store
	auth    *app.Auth
	users   *app.Users
	clients *app.Clients
	notif   *app.Notifications
	share   *app.Share
	clock   *clock.Fixed
	admin   *domain.User
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, identity.Migrations(), "migrations", identity.Schema)
	pg.TruncateSchema(t, pool, identity.Schema)
	store := postgres.NewStore(pool)
	// Real "now": issued tokens are checked by a verifier that reads the wall clock.
	clk := &clock.Fixed{T: time.Now().UTC().Truncate(time.Second)}
	priv, _ := authn.KeyPair("", "test-secret")
	f := &fixture{
		store:   store,
		auth:    app.NewAuth(store, authn.NewIssuer(priv, 15*time.Minute), clk),
		users:   app.NewUsers(store, events.Noop{}),
		clients: app.NewClients(store, events.Noop{}),
		notif:   app.NewNotifications(store),
		share:   app.NewShare(store, "test-secret", clk),
		clock:   clk,
	}
	admin, err := f.users.Create(context.Background(), app.UserInput{Name: "Admin", Email: "Admin@White.id ", Password: "admin12345", Role: domain.RoleAdmin, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	f.admin = admin
	return f
}

func code(t *testing.T, err error, want apperr.Code) *apperr.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", want)
	}
	e := apperr.As(err)
	if e.Code != want {
		t.Fatalf("expected %s, got %s (%s)", want, e.Code, e.Message)
	}
	return e
}

func TestIntegrationLoginRefreshDeactivate(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	if _, err := f.auth.Login(ctx, "admin@white.id", "wrong", "", ""); err == nil {
		t.Fatal("wrong password accepted")
	}
	tok, err := f.auth.Login(ctx, "ADMIN@white.id", "admin12345", "ua", "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if tok.User.Email != "admin@white.id" || tok.AccessToken == "" || tok.RefreshToken == "" {
		t.Fatalf("%+v", tok)
	}
	pub, _ := authn.PublicKey("", "", "test-secret")
	p, err := authn.NewVerifier(pub).Verify(tok.AccessToken)
	if err != nil || p.UserID != f.admin.ID || !p.IsAdmin() {
		t.Fatalf("%+v %v", p, err)
	}

	rotated, err := f.auth.Refresh(ctx, tok.RefreshToken, "", "")
	if err != nil {
		t.Fatal(err)
	}
	code(t, errOf(f.auth.Refresh(ctx, tok.RefreshToken, "", "")), apperr.Unauthenticated)

	member, err := f.users.Create(ctx, app.UserInput{Name: "Tim", Email: "tim@white.id", Password: "member12345", Role: domain.RoleMember, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	mt, err := f.auth.Login(ctx, "tim@white.id", "member12345", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.users.SetActive(ctx, f.admin.ID, member.ID, false); err != nil {
		t.Fatal(err)
	}
	code(t, errOf(f.auth.Refresh(ctx, mt.RefreshToken, "", "")), apperr.Unauthenticated)
	code(t, errOf(f.auth.Login(ctx, "tim@white.id", "member12345", "", "")), apperr.Unauthenticated)
	code(t, errOf(f.auth.Me(ctx, member.ID)), apperr.Unauthenticated)

	if _, err := f.auth.Me(ctx, rotated.User.ID); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationUsersRules(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	e := code(t, errOf(f.users.Create(ctx, app.UserInput{Name: "", Email: "bad", Password: "short", Role: "X", IsActive: true})), apperr.Validation)
	for _, field := range []string{"name", "email", "password", "role"} {
		if len(e.FieldErrors[field]) == 0 {
			t.Fatalf("missing field error for %s: %+v", field, e.FieldErrors)
		}
	}
	e = code(t, errOf(f.users.Create(ctx, app.UserInput{Name: "Dup", Email: "admin@white.id", Password: "12345678", Role: domain.RoleMember, IsActive: true})), apperr.Validation)
	if e.FieldErrors["email"][0] != "Email sudah terdaftar" {
		t.Fatalf("%+v", e.FieldErrors)
	}

	code(t, errOf(f.users.Update(ctx, f.admin.ID, f.admin.ID, app.UserInput{Name: "Admin", Email: "admin@white.id", Role: domain.RoleMember, IsActive: true})), apperr.Validation)
	code(t, errOf(f.users.SetActive(ctx, f.admin.ID, f.admin.ID, false)), apperr.Validation)
	code(t, f.users.Delete(ctx, f.admin.ID, f.admin.ID), apperr.Validation)

	c1, _ := f.clients.Create(ctx, f.admin.ID, app.ClientProfile{Name: "A", Slug: "aa", Currency: "IDR", Timezone: "Asia/Jakarta"}, nil)
	c2, _ := f.clients.Create(ctx, f.admin.ID, app.ClientProfile{Name: "B", Slug: "bb", Currency: "IDR", Timezone: "Asia/Jakarta"}, nil)
	u, err := f.users.Create(ctx, app.UserInput{Name: "Tim", Email: "tim@white.id", Password: "member12345", Role: domain.RoleMember, IsActive: true,
		Memberships: []domain.Membership{{ClientID: c1.ID, Role: domain.MemberViewer}}})
	if err != nil {
		t.Fatal(err)
	}
	// Update replaces every membership and keeps the password when blank.
	if _, err := f.users.Update(ctx, f.admin.ID, u.ID, app.UserInput{Name: "Tim", Email: "tim@white.id", Role: domain.RoleMember, IsActive: true,
		Memberships: []domain.Membership{{ClientID: c2.ID, Role: domain.MemberManager}}}); err != nil {
		t.Fatal(err)
	}
	list, _ := f.users.List(ctx)
	var tim *app.UserWithClients
	for i := range list {
		if list[i].User.ID == u.ID {
			tim = &list[i]
		}
	}
	if tim == nil || len(tim.Clients) != 1 || tim.Clients[0].ClientID != c2.ID || tim.Clients[0].Role != domain.MemberManager {
		t.Fatalf("%+v", tim)
	}
	if _, err := f.auth.Login(ctx, "tim@white.id", "member12345", "", ""); err != nil {
		t.Fatal("password changed on blank update:", err)
	}
	if err := f.users.Delete(ctx, f.admin.ID, u.ID); err != nil {
		t.Fatal(err)
	}
	code(t, f.users.Delete(ctx, f.admin.ID, u.ID), apperr.NotFound)
}

func TestIntegrationClientsAccessShare(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	e := code(t, errOf(f.clients.Create(ctx, f.admin.ID, app.ClientProfile{Name: "", Slug: "Bad Slug", WebsiteURL: "ftp://x", Currency: "XXX", Timezone: "Mars"}, nil)), apperr.Validation)
	for _, field := range []string{"name", "slug", "websiteUrl", "currency", "timezone"} {
		if len(e.FieldErrors[field]) == 0 {
			t.Fatalf("missing %s: %+v", field, e.FieldErrors)
		}
	}
	c, err := f.clients.Create(ctx, f.admin.ID, app.ClientProfile{Name: "Kopi", Slug: " KOPI ", WebsiteURL: "https://kopi.id", Currency: "IDR", Timezone: "Asia/Jakarta"}, []string{"SEO"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Slug != "kopi" || len(c.ShareModules) != 1 || c.ShareModules[0] != "SEO" {
		t.Fatalf("%+v", c)
	}
	e = code(t, errOf(f.clients.Create(ctx, f.admin.ID, app.ClientProfile{Name: "Dup", Slug: "kopi", Currency: "IDR", Timezone: "Asia/Jakarta"}, nil)), apperr.Validation)
	if e.FieldErrors["slug"][0] != "Slug sudah dipakai" {
		t.Fatalf("%+v", e.FieldErrors)
	}

	viewer, _ := f.users.Create(ctx, app.UserInput{Name: "V", Email: "v@white.id", Password: "12345678", Role: domain.RoleMember, IsActive: true})
	code(t, errOf(f.clients.Access(ctx, viewer.ID, false, c.ID, "")), apperr.Unauthorized)
	code(t, errOf(f.clients.Access(ctx, viewer.ID, false, "", "missing")), apperr.NotFound)
	if err := f.clients.UpsertMember(ctx, f.admin.ID, true, c.ID, viewer.ID, domain.MemberViewer); err != nil {
		t.Fatal(err)
	}
	acc, err := f.clients.Access(ctx, viewer.ID, false, "", "kopi")
	if err != nil || acc.CanManage || acc.Role != domain.AccessViewer {
		t.Fatalf("%+v %v", acc, err)
	}
	// Viewer cannot manage; unknown client reports UNAUTHORIZED like manageContext did.
	code(t, errOf(f.clients.Update(ctx, viewer.ID, false, c.ID, app.ClientProfile{Name: "X", Slug: "kopi", Currency: "IDR", Timezone: "Asia/Jakarta"})), apperr.Unauthorized)
	code(t, errOf(f.clients.Update(ctx, f.admin.ID, true, "nope", app.ClientProfile{Name: "X", Slug: "kopi", Currency: "IDR", Timezone: "Asia/Jakarta"})), apperr.Unauthorized)

	summaries, _ := f.clients.List(ctx, viewer.ID, false)
	if len(summaries) != 1 || summaries[0].MembersCount != 1 || summaries[0].MyRole != domain.AccessViewer {
		t.Fatalf("%+v", summaries)
	}

	// Share: enable with a PIN, verify lock/unlock, rate limit, then clear the PIN.
	if _, err := f.clients.UpdateShare(ctx, f.admin.ID, true, c.ID, true, "123456", []string{"SOCIAL", "SEO"}); err != nil {
		t.Fatal(err)
	}
	sc, _ := f.share.Context(ctx, "kopi", "")
	if sc.Status != app.ShareLocked {
		t.Fatalf("%+v", sc)
	}
	code(t, errOf2(f.share.VerifyPin(ctx, "kopi", "000000", "9.9.9.9")), apperr.Validation)
	_, token, err := f.share.VerifyPin(ctx, "kopi", "123456", "9.9.9.9")
	if err != nil || token == "" {
		t.Fatal(token, err)
	}
	if sc, _ = f.share.Context(ctx, "kopi", token); sc.Status != app.ShareOK {
		t.Fatalf("%+v", sc)
	}
	for i := 0; i < 3; i++ {
		_, _, _ = f.share.VerifyPin(ctx, "kopi", "000000", "9.9.9.9")
	}
	code(t, errOf2(f.share.VerifyPin(ctx, "kopi", "123456", "9.9.9.9")), apperr.RateLimit)
	f.clock.T = f.clock.T.Add(11 * time.Minute)
	if _, _, err := f.share.VerifyPin(ctx, "kopi", "123456", "9.9.9.9"); err != nil {
		t.Fatal("window did not reset:", err)
	}
	if _, err := f.clients.UpdateShare(ctx, f.admin.ID, true, c.ID, true, "", []string{"SEO"}); err != nil {
		t.Fatal(err)
	}
	if sc, _ = f.share.Context(ctx, "kopi", ""); sc.Status != app.ShareOK {
		t.Fatalf("empty PIN must clear the lock: %+v", sc)
	}
	code(t, errOf(f.clients.UpdateShare(ctx, f.admin.ID, true, c.ID, true, "12", []string{"SEO"})), apperr.Validation)
	code(t, errOf(f.clients.UpdateShare(ctx, f.admin.ID, true, c.ID, true, "", nil)), apperr.Validation)
}

func TestIntegrationNotifyAudience(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c, _ := f.clients.Create(ctx, f.admin.ID, app.ClientProfile{Name: "A", Slug: "aa", Currency: "IDR", Timezone: "Asia/Jakarta"}, nil)
	mgr, _ := f.users.Create(ctx, app.UserInput{Name: "M", Email: "m@white.id", Password: "12345678", Role: domain.RoleMember, IsActive: true, Memberships: []domain.Membership{{ClientID: c.ID, Role: domain.MemberManager}}})
	viewer, _ := f.users.Create(ctx, app.UserInput{Name: "V", Email: "v@white.id", Password: "12345678", Role: domain.RoleMember, IsActive: true, Memberships: []domain.Membership{{ClientID: c.ID, Role: domain.MemberViewer}}})
	inactive, _ := f.users.Create(ctx, app.UserInput{Name: "I", Email: "i@white.id", Password: "12345678", Role: domain.RoleMember, IsActive: false, Memberships: []domain.Membership{{ClientID: c.ID, Role: domain.MemberManager}}})
	_ = inactive

	n, err := f.notif.Notify(ctx, app.NotifyInput{ClientID: c.ID, Audience: app.AudienceClientApprovers, ExceptUserID: mgr.ID, Type: "POST_REVIEW", Title: "Review"})
	if err != nil || n != 1 { // admin only: manager excluded, viewer not an approver, inactive skipped
		t.Fatalf("n=%d err=%v", n, err)
	}
	n, err = f.notif.Notify(ctx, app.NotifyInput{ClientID: c.ID, Audience: app.AudienceClientAll, UserIDs: []string{viewer.ID, viewer.ID}, Type: "POST_PUBLISHED", Title: "Terbit", Href: "/x"})
	if err != nil || n != 3 { // admin, manager, viewer (deduped)
		t.Fatalf("n=%d err=%v", n, err)
	}
	items, unread, _ := f.notif.Feed(ctx, viewer.ID, 20)
	if unread != 1 || len(items) != 1 || items[0].Href == nil || *items[0].Href != "/x" {
		t.Fatalf("%d %+v", unread, items)
	}
	_ = f.notif.MarkRead(ctx, f.admin.ID, items[0].ID) // someone else's id: silent no-op
	if _, unread, _ = f.notif.Feed(ctx, viewer.ID, 20); unread != 1 {
		t.Fatal("foreign mark-read changed state")
	}
	if cnt, _ := f.notif.MarkAllRead(ctx, viewer.ID); cnt != 1 {
		t.Fatal(cnt)
	}
}

func errOf[T any](_ T, err error) error { return err }

func errOf2[A, B any](_ A, _ B, err error) error { return err }

func TestIntegrationPermissionsTravelInToken(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	code(t, errOf(f.users.Create(ctx, app.UserInput{Name: "X", Email: "x@white.id", Password: "12345678", Role: domain.RoleMember, IsActive: true, Permissions: []string{"root.all"}})), apperr.Validation)
	u, err := f.users.Create(ctx, app.UserInput{Name: "Fin", Email: "fin@white.id", Password: "12345678", Role: domain.RoleMember, IsActive: true, Permissions: []string{authn.PermFinanceRead, "hr.*"}})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := f.auth.Login(ctx, "fin@white.id", "12345678", "", "")
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := authn.PublicKey("", "", "test-secret")
	p, err := authn.NewVerifier(pub).Verify(tok.AccessToken)
	if err != nil || !p.Can(authn.PermFinanceRead) || p.Can(authn.PermFinanceWrite) || !p.Can(authn.PermPayroll) {
		t.Fatalf("%+v %v", p, err)
	}
	// Update replaces the set.
	if _, err := f.users.Update(ctx, f.admin.ID, u.ID, app.UserInput{Name: "Fin", Email: "fin@white.id", Role: domain.RoleMember, IsActive: true, Permissions: []string{authn.PermSalesRead}}); err != nil {
		t.Fatal(err)
	}
	got, _ := f.store.GetUser(ctx, u.ID)
	if len(got.Permissions) != 1 || got.Permissions[0] != authn.PermSalesRead {
		t.Fatalf("%+v", got.Permissions)
	}
}

func TestIntegrationLegacySyncAndPermissions(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	hash, _ := password.Hash("legacy-pass")
	u, err := f.users.SyncLegacy(ctx, domain.User{ID: "cm-legacy-1", Email: " New@White.ID", Name: "Baru", PasswordHash: hash, Role: domain.RoleMember, IsActive: true})
	if err != nil || u.Email != "new@white.id" {
		t.Fatalf("%+v %v", u, err)
	}
	if _, err := f.users.SetPermissions(ctx, u.ID, []string{authn.PermSalesRead}); err != nil {
		t.Fatal(err)
	}
	// A later sync (rename, deactivate) keeps permissions and blocks login.
	if _, err := f.users.SyncLegacy(ctx, domain.User{ID: "cm-legacy-1", Email: "new@white.id", Name: "Baru Lagi", PasswordHash: hash, Role: domain.RoleMember, IsActive: false}); err != nil {
		t.Fatal(err)
	}
	got, _ := f.store.GetUser(ctx, "cm-legacy-1")
	if got.Name != "Baru Lagi" || len(got.Permissions) != 1 || got.IsActive {
		t.Fatalf("%+v", got)
	}
	code(t, errOf(f.auth.Login(ctx, "new@white.id", "legacy-pass", "", "")), apperr.Unauthenticated)
	code(t, errOf(f.users.SetPermissions(ctx, u.ID, []string{"nope.x"})), apperr.Validation)
	code(t, errOf(f.users.SetPermissions(ctx, "missing", nil)), apperr.NotFound)
	if err := f.users.RemoveLegacy(ctx, "cm-legacy-1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.store.GetUser(ctx, "cm-legacy-1"); got != nil {
		t.Fatal("not removed")
	}
}

func TestIntegrationPermissionHolders(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	mk := func(email string, active bool, perms ...string) {
		if _, err := f.users.Create(ctx, app.UserInput{Name: email, Email: email, Password: "12345678", Role: domain.RoleMember, IsActive: active, Permissions: perms}); err != nil {
			t.Fatal(err)
		}
	}
	mk("direct@white.id", true, authn.PermFinanceApprove)
	mk("wild@white.id", true, "finance.*")
	mk("other@white.id", true, authn.PermFinanceRead)
	mk("off@white.id", false, authn.PermFinanceApprove)

	got, err := f.users.PermissionHolders(ctx, authn.PermFinanceApprove)
	if err != nil {
		t.Fatal(err)
	}
	emails := map[string]bool{}
	for _, u := range got {
		emails[u.Email] = true
	}
	if len(got) != 3 || !emails["direct@white.id"] || !emails["wild@white.id"] || !emails[f.admin.Email] {
		t.Fatalf("%v", emails)
	}
	code(t, errOf(f.users.PermissionHolders(ctx, "root.all")), apperr.Validation)
}
