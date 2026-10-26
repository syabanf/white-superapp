package app

import (
	"context"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
)

type postsOnly struct{ posts []domain.SyncedPost }

func (p postsOnly) OwnAccounts(context.Context, string) ([]domain.SocialAccount, error) {
	return nil, nil
}

func (p postsOnly) SyncedPosts(context.Context, string, []string, int32) ([]domain.SyncedPost, error) {
	return p.posts, nil
}

func TestAttachEngagementsMatchesAccountAndExternalID(t *testing.T) {
	ext := func(s string) *string { return &s }
	targets := []domain.PublishedTarget{
		{PostID: "p1", SocialAccountID: "ig", ExternalID: ext("111")},
		{PostID: "p2", SocialAccountID: "fb", ExternalID: ext("111")},
		{PostID: "p3", SocialAccountID: "ig"},
	}
	r := &Reports{Deps{Analytics: postsOnly{posts: []domain.SyncedPost{
		{SocialAccountID: "ig", ExternalID: "111", PublishedAt: time.Now(), Engagements: 42},
	}}}}
	if err := r.attachEngagements(context.Background(), "c1", targets); err != nil {
		t.Fatal(err)
	}
	if targets[0].Engagements == nil || *targets[0].Engagements != 42 {
		t.Fatalf("ig target not matched: %+v", targets[0])
	}
	if targets[1].Engagements != nil || targets[2].Engagements != nil {
		t.Fatal("matched the wrong account or a target without an external id")
	}
	if s := domain.SummarizePublishing(targets, 0, 5); s.Mode != "engagement" {
		t.Fatalf("mode = %q", s.Mode)
	}
}
