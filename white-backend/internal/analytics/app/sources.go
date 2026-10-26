package app

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/metagraph"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

const (
	msgUsernameRequired = "Nama klien wajib diisi"
	msgNameRequired     = "Nama klien wajib diisi"
	msgWebsiteInvalid   = "URL website tidak valid (awali dengan https://)"
	msgDuplicate        = "Sudah terdaftar untuk klien ini"
	msgUsernameInvalid  = "Username tidak valid. Gunakan 2–30 karakter huruf, angka, titik, atau garis bawah."
	msgCompetitorExists = "Kompetitor dengan username ini sudah ada."

	competitorHistoryDays = 120
)

var scDomainRe = regexp.MustCompile(`(?i)^sc-domain:[a-z0-9.-]+$`)

type Sources struct{ Deps }

func NewSources(d Deps) *Sources { return &Sources{d} }

// AccountWithFollowers is a social account with its latest snapshot followers.
type AccountWithFollowers struct {
	domain.SocialAccount
	Followers int
}

func (s *Sources) withFollowers(ctx context.Context, accounts []domain.SocialAccount) ([]AccountWithFollowers, error) {
	idList := make([]string, 0, len(accounts))
	for _, a := range accounts {
		idList = append(idList, a.ID)
	}
	latest := map[string]int{}
	if len(idList) > 0 {
		var err error
		if latest, err = s.Store.LatestFollowers(ctx, idList); err != nil {
			return nil, err
		}
	}
	out := make([]AccountWithFollowers, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, AccountWithFollowers{SocialAccount: a, Followers: latest[a.ID]})
	}
	return out, nil
}

// ListSocialAccounts is used by the UI (checks access when userID is set)
// and by publishing over the internal path.
func (s *Sources) ListSocialAccounts(ctx context.Context, userID, clientID string, includeCompetitors bool, platform *domain.Platform) ([]AccountWithFollowers, error) {
	if userID != "" {
		if _, err := s.view(ctx, userID, clientID); err != nil {
			return nil, err
		}
	}
	accounts, err := s.Store.ListSocialAccounts(ctx, clientID, includeCompetitors, platform)
	if err != nil {
		return nil, err
	}
	return s.withFollowers(ctx, accounts)
}

func (s *Sources) ListSeoProperties(ctx context.Context, userID, clientID string) ([]domain.SeoProperty, error) {
	if userID != "" {
		if _, err := s.view(ctx, userID, clientID); err != nil {
			return nil, err
		}
	}
	return s.Store.ListSeoProperties(ctx, clientID)
}

func (s *Sources) ListAdAccounts(ctx context.Context, userID, clientID string) ([]domain.AdAccount, error) {
	if userID != "" {
		if _, err := s.view(ctx, userID, clientID); err != nil {
			return nil, err
		}
	}
	return s.Store.ListAdAccounts(ctx, clientID)
}

// GetSeoProperty is internal (seosuite). Missing is NOT_FOUND.
func (s *Sources) GetSeoProperty(ctx context.Context, id string) (*domain.SeoProperty, error) {
	p, err := s.Store.GetSeoProperty(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperr.New(apperr.NotFound, msgNoProperty)
	}
	return p, nil
}

// ListSocialPosts is internal (publishing): own accounts only, newest first.
func (s *Sources) ListSocialPosts(ctx context.Context, f domain.PostFilter, fromDay, toDay string) ([]domain.SocialPost, error) {
	if f.ClientID == "" {
		return nil, apperr.New(apperr.Validation, "")
	}
	if fromDay != "" {
		d, err := dates.Parse(fromDay)
		if err != nil {
			return nil, apperr.Field("fromDay", apperr.Message(apperr.Validation))
		}
		f.From = &d
	}
	if toDay != "" {
		d, err := dates.Parse(toDay)
		if err != nil {
			return nil, apperr.Field("toDay", apperr.Message(apperr.Validation))
		}
		end := endExclusive(d)
		f.ToExclusive = &end
	}
	if f.Limit <= 0 {
		f.Limit = 400
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}
	return s.Store.ListOwnPostsFiltered(ctx, f)
}

// ClientSources is the settings page view: own social accounts by
// platform/username, properties by site URL, ad accounts by name.
type ClientSources struct {
	Social []AccountWithFollowers
	Seo    []domain.SeoProperty
	Ads    []domain.AdAccount
}

func (s *Sources) GetClientSources(ctx context.Context, userID, clientID string) (*ClientSources, error) {
	if _, err := s.view(ctx, userID, clientID); err != nil {
		return nil, err
	}
	accounts, err := s.Store.ListSocialAccounts(ctx, clientID, false, nil)
	if err != nil {
		return nil, err
	}
	social, err := s.withFollowers(ctx, accounts)
	if err != nil {
		return nil, err
	}
	sortBy(social, func(a, b AccountWithFollowers) bool {
		if a.Platform != b.Platform {
			return a.Platform < b.Platform
		}
		return a.Username < b.Username
	})
	props, err := s.Store.ListSeoProperties(ctx, clientID)
	if err != nil {
		return nil, err
	}
	sortBy(props, func(a, b domain.SeoProperty) bool { return a.SiteURL < b.SiteURL })
	ads, err := s.Store.ListAdAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	sortBy(ads, func(a, b domain.AdAccount) bool { return a.Name < b.Name })
	return &ClientSources{Social: social, Seo: props, Ads: ads}, nil
}

func (s *Sources) connection(ctx context.Context, clientID, provider string) (*string, error) {
	id, err := s.Integ.FindClientConnection(ctx, clientID, provider)
	if err != nil {
		return nil, err
	}
	return validate.EmptyToNil(id), nil
}

func isUnique(err error) bool {
	var u *domain.ErrUnique
	return errors.As(err, &u)
}

func (s *Sources) AddSocialAccount(ctx context.Context, userID, clientID string, platform domain.Platform, username, displayName string) (*AccountWithFollowers, error) {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	v := validate.New()
	if !domain.IsPlatform(platform) {
		v.Add("platform", apperr.Message(apperr.Validation))
	}
	v.MinLen("username", username, 1, msgUsernameRequired).MaxLen("username", username, 60, msgTooLong)
	v.MaxLen("displayName", displayName, 80, msgTooLong)
	if err := v.Err(); err != nil {
		return nil, err
	}
	username = strings.ToLower(strings.TrimLeft(username, "@"))
	if displayName == "" {
		displayName = username
	}
	conn, err := s.connection(ctx, clientID, domain.IntegrationMeta)
	if err != nil {
		return nil, err
	}
	a, err := s.Store.InsertSocialAccount(ctx, domain.SocialAccount{ID: ids.New(), ClientID: clientID, Platform: platform, ExternalID: strings.ToLower(string(platform)) + ":" + username, Username: username, DisplayName: displayName, ConnectionID: conn})
	if err != nil {
		if isUnique(err) {
			return nil, apperr.Field("username", msgDuplicate)
		}
		return nil, err
	}
	return &AccountWithFollowers{SocialAccount: *a}, nil
}

func (s *Sources) RemoveSocialAccount(ctx context.Context, userID, clientID, accountID string) error {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return err
	}
	a, err := s.Store.GetSocialAccount(ctx, accountID)
	if err != nil {
		return err
	}
	if a == nil || a.ClientID != clientID || a.IsCompetitor {
		return apperr.New(apperr.NotFound, "")
	}
	_, err = s.Store.DeleteSocialAccount(ctx, accountID)
	return err
}

func (s *Sources) AddSeoProperty(ctx context.Context, userID, clientID, siteURL, ga4 string) (*domain.SeoProperty, error) {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	siteURL = strings.TrimSpace(siteURL)
	ga4 = strings.TrimSpace(ga4)
	v := validate.New()
	if utf8.RuneCountInString(siteURL) < 4 || !(scDomainRe.MatchString(siteURL) || validate.IsHTTPURL(siteURL)) {
		v.Add("siteUrl", msgWebsiteInvalid)
	}
	v.MaxLen("ga4PropertyId", ga4, 30, msgTooLong)
	if err := v.Err(); err != nil {
		return nil, err
	}
	conn, err := s.connection(ctx, clientID, domain.IntegrationGoogle)
	if err != nil {
		return nil, err
	}
	p, err := s.Store.InsertSeoProperty(ctx, domain.SeoProperty{ID: ids.New(), ClientID: clientID, SiteURL: siteURL, Ga4PropertyID: validate.EmptyToNil(ga4), ConnectionID: conn})
	if err != nil {
		if isUnique(err) {
			return nil, apperr.Field("siteUrl", msgDuplicate)
		}
		return nil, err
	}
	return p, nil
}

func (s *Sources) RemoveSeoProperty(ctx context.Context, userID, clientID, propertyID string) error {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return err
	}
	p, err := s.Store.GetSeoProperty(ctx, propertyID)
	if err != nil {
		return err
	}
	if p == nil || p.ClientID != clientID {
		return apperr.New(apperr.NotFound, "")
	}
	_, err = s.Store.DeleteSeoProperty(ctx, propertyID)
	return err
}

// SetAuditCadence is allowed for any member (old guard had no role check).
func (s *Sources) SetAuditCadence(ctx context.Context, userID, clientID, propertyID string, c domain.AuditCadence) (domain.AuditCadence, error) {
	if !domain.IsCadence(c) {
		return "", apperr.New(apperr.Validation, "")
	}
	p, err := s.seoGuard(ctx, userID, clientID, propertyID)
	if err != nil {
		return "", err
	}
	return c, s.Store.SetAuditCadence(ctx, p.ID, c)
}

func (s *Sources) AddAdAccount(ctx context.Context, userID, clientID, externalID, name string) (*domain.AdAccount, error) {
	acc, err := s.manage(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	externalID = strings.TrimSpace(externalID)
	name = strings.TrimSpace(name)
	v := validate.New()
	v.MinLen("externalId", externalID, 2, apperr.Message(apperr.Validation)).MaxLen("externalId", externalID, 40, msgTooLong)
	v.MinLen("name", name, 1, msgNameRequired).MaxLen("name", name, 80, msgTooLong)
	if err := v.Err(); err != nil {
		return nil, err
	}
	conn, err := s.connection(ctx, clientID, domain.IntegrationMeta)
	if err != nil {
		return nil, err
	}
	currency := acc.Currency
	if currency == "" {
		currency = "IDR"
	}
	a, err := s.Store.InsertAdAccount(ctx, domain.AdAccount{ID: ids.New(), ClientID: clientID, ExternalID: externalID, Name: name, Currency: currency, ConnectionID: conn})
	if err != nil {
		if isUnique(err) {
			return nil, apperr.Field("externalId", msgDuplicate)
		}
		return nil, err
	}
	return a, nil
}

func (s *Sources) RemoveAdAccount(ctx context.Context, userID, clientID, accountID string) error {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return err
	}
	a, err := s.Store.GetAdAccount(ctx, accountID)
	if err != nil {
		return err
	}
	if a == nil || a.ClientID != clientID {
		return apperr.New(apperr.NotFound, "")
	}
	_, err = s.Store.DeleteAdAccount(ctx, accountID)
	return err
}

// ─── Competitors ─────────────────────────────────────────────

// discovered is the competitor profile from Apify or Business Discovery.
type discovered struct {
	username, name string
	biography      *string
	avatar         *string
	followers      int
	mediaCount     int
	media          []domain.SocialPost
}

func (s *Sources) discover(ctx context.Context, clientID, username string) (*discovered, error) {
	if s.Syncer.ApifyConfigured(ctx) {
		pub := s.Syncer.PublicSocial(ctx)
		profile, err := pub.Profile(ctx, string(domain.Instagram), username)
		if err != nil {
			return nil, err
		}
		posts, err := pub.Posts(ctx, string(domain.Instagram), username, 20)
		if err != nil {
			return nil, err
		}
		d := &discovered{username: profile.Username, name: profile.DisplayName, biography: profile.Biography, avatar: profile.AvatarURL, followers: profile.Followers, mediaCount: profile.MediaCount}
		for _, p := range posts {
			d.media = append(d.media, domain.SocialPost{ExternalID: p.ExternalID, Caption: p.Caption, MediaType: p.MediaType, ProductType: p.ProductType, Permalink: p.Permalink, MediaURL: p.MediaURL, ThumbnailURL: p.ThumbnailURL, PublishedAt: p.PublishedAt, Likes: p.Likes, Comments: p.Comments})
		}
		return d, nil
	}
	// Business Discovery through the own IG account. Real only with app
	// credentials and a token of that account; otherwise the mock.
	ig := domain.Instagram
	own, err := s.Store.ListSocialAccounts(ctx, clientID, false, &ig)
	if err != nil {
		return nil, err
	}
	externalID, token := "", ""
	if len(own) > 0 {
		externalID = own[0].ExternalID
		if own[0].ConnectionID != nil {
			if token, err = s.Integ.AccessToken(ctx, *own[0].ConnectionID); err != nil {
				return nil, err
			}
		}
	}
	graph := s.Syncer.Graph
	if graph == nil {
		graph = metagraph.Pick(token != "" && s.Syncer.MetaConfigured(ctx))
	}
	bd, err := graph.DiscoverBusiness(ctx, token, externalID, username)
	if err != nil {
		return nil, err
	}
	d := &discovered{username: bd.Username, name: bd.Name, biography: bd.Biography, avatar: bd.ProfilePictureURL, followers: bd.FollowersCount, mediaCount: bd.MediaCount}
	for _, m := range bd.Media {
		d.media = append(d.media, domain.SocialPost{ExternalID: m.ID, Caption: m.Caption, MediaType: m.MediaType, ProductType: m.MediaProductType, Permalink: m.Permalink, MediaURL: m.MediaURL, ThumbnailURL: m.ThumbnailURL, PublishedAt: m.Timestamp, Likes: m.LikeCount, Comments: m.CommentsCount})
	}
	return d, nil
}

// AddCompetitor adds an Instagram competitor with 120 days of synthetic
// snapshots ending at the discovered follower count and up to 20 posts.
func (s *Sources) AddCompetitor(ctx context.Context, userID, clientID, rawUsername string) (accountID, username string, err error) {
	if n := utf8.RuneCountInString(rawUsername); n < 1 || n > 64 || clientID == "" {
		return "", "", apperr.New(apperr.Validation, "")
	}
	username = domain.NormalizeIgUsername(rawUsername)
	if username == "" {
		return "", "", apperr.New(apperr.Validation, msgUsernameInvalid)
	}
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return "", "", err
	}
	existing, err := s.Store.FindCompetitor(ctx, clientID, username)
	if err != nil {
		return "", "", err
	}
	if existing != nil {
		return "", "", apperr.New(apperr.Validation, msgCompetitorExists)
	}
	d, err := s.discover(ctx, clientID, username)
	if err != nil {
		return "", "", providerFail(err)
	}
	acc, err := s.Store.InsertSocialAccount(ctx, domain.SocialAccount{ID: ids.New(), ClientID: clientID, Platform: domain.Instagram, ExternalID: "ig_bd_" + username, Username: d.username, DisplayName: d.name, Biography: d.biography, AvatarURL: d.avatar, IsCompetitor: true})
	if err != nil {
		if isUnique(err) {
			return "", "", apperr.New(apperr.Validation, msgCompetitorExists)
		}
		return "", "", apperr.Wrap(apperr.Unknown, "", err)
	}
	end := dates.LatestCompleteDay(s.now())
	series := buildSeries(username, d.followers)
	snaps := make([]domain.SocialSnapshot, 0, len(series))
	for i, f := range series {
		back := len(series) - 1 - i
		snaps = append(snaps, domain.SocialSnapshot{AccountID: acc.ID, Date: dates.AddDays(end, -back), Followers: f, MediaCount: int(math.Max(0, float64(d.mediaCount)-math.Floor(float64(back)*0.3+0.5)))})
	}
	if err := s.Store.InsertSnapshotsIgnore(ctx, snaps); err != nil {
		return "", "", apperr.Wrap(apperr.Unknown, "", err)
	}
	media := d.media
	if len(media) > 20 {
		media = media[:20]
	}
	for i := range media {
		media[i].AccountID = acc.ID
	}
	if err := s.Store.InsertPostsIgnore(ctx, media); err != nil {
		return "", "", apperr.Wrap(apperr.Unknown, "", err)
	}
	return acc.ID, d.username, nil
}

func buildSeries(username string, followers int) []int {
	return metagraph.BuildFollowerSeries("bd:"+username, competitorHistoryDays, followers)
}

// providerFail mirrors fail(t.errors[code], code) for provider errors and
// UNKNOWN for anything else.
func providerFail(err error) error {
	var e *apperr.Error
	if errors.As(err, &e) && e.Provider != "" {
		out := apperr.New(e.Code, "")
		out.Provider = e.Provider
		out.RequiresReconnect = e.RequiresReconnect
		return out
	}
	return apperr.Wrap(apperr.Unknown, "", err)
}

func (s *Sources) RemoveCompetitor(ctx context.Context, userID, clientID, accountID string) error {
	if accountID == "" {
		return apperr.New(apperr.Validation, "")
	}
	a, err := s.Store.GetSocialAccount(ctx, accountID)
	if err != nil {
		return err
	}
	if a == nil || !a.IsCompetitor || a.ClientID != clientID {
		return apperr.New(apperr.NotFound, "")
	}
	if _, err := s.manage(ctx, userID, a.ClientID); err != nil {
		return err
	}
	_, err = s.Store.DeleteSocialAccount(ctx, accountID)
	return err
}

// ─── Lifecycle events ────────────────────────────────────────

// OnClientCreated creates the SEO property when SEO is on and a website is set.
func (s *Sources) OnClientCreated(ctx context.Context, clientID, websiteURL string, modules []string) error {
	websiteURL = strings.TrimSpace(websiteURL)
	if websiteURL == "" || !contains(modules, "SEO") {
		return nil
	}
	_, err := s.Store.InsertSeoProperty(ctx, domain.SeoProperty{ID: ids.New(), ClientID: clientID, SiteURL: websiteURL})
	if isUnique(err) {
		return nil
	}
	return err
}

func (s *Sources) PurgeClient(ctx context.Context, clientID string) error {
	return s.Store.WithTx(ctx, func(tx domain.Store) error { return tx.PurgeClient(ctx, clientID) })
}

// RetentionDays of sync jobs.
const RetentionDays = 90

func (s *Sources) RunRetention(ctx context.Context) (int64, error) {
	return s.Store.DeleteSyncJobsBefore(ctx, s.now().AddDate(0, 0, -RetentionDays))
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
