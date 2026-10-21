// Package publisher sends posts to Instagram and Facebook pages through the
// Meta Graph API, with a deterministic mock for demo mode. Both satisfy
// domain.Publisher; Pick chooses per target.
package publisher

import (
	"context"
	"strconv"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// MockFailMarker in a caption makes the mock fail so the retry flow can be
// exercised without credentials.
const MockFailMarker = "[fail]"

type Mock struct{}

func (Mock) Publish(_ context.Context, in domain.PublishInput) (*domain.PublishResult, error) {
	if strings.Contains(in.Text, MockFailMarker) {
		return nil, apperr.FromProvider("mock-publisher", apperr.Unknown, "Publikasi gagal (simulasi — caption mengandung [fail])", nil)
	}
	urls := make([]string, 0, len(in.Media))
	for _, m := range in.Media {
		urls = append(urls, m.URL)
	}
	seed := mockrand.Seed(string(in.Platform) + ":" + in.AccountExternalID + ":" + in.Text + ":" + strings.Join(urls, ","))
	externalID := "mock_" + strings.ToLower(string(in.Platform)) + "_" + strconv.FormatUint(uint64(seed), 36) + strconv.FormatUint(uint64(seed%997), 36)
	permalink := MockPermalink(in.Platform, in.AccountExternalID, externalID)
	return &domain.PublishResult{ExternalID: externalID, Permalink: &permalink}, nil
}

func MockPermalink(platform domain.Platform, accountExternalID, externalID string) string {
	switch platform {
	case domain.PlatformInstagram:
		tail := externalID
		if len(tail) > 11 {
			tail = tail[len(tail)-11:]
		}
		return "https://www.instagram.com/p/" + tail + "/"
	case domain.PlatformFacebook:
		return "https://www.facebook.com/" + accountExternalID + "/posts/" + externalID
	}
	return "https://www.tiktok.com/@" + accountExternalID + "/video/" + externalID
}

// Pick returns the real publisher only when the Meta app is configured AND
// the account has a connection; otherwise the mock (scheduler.ts useReal).
func Pick(real, mock domain.Publisher, metaConfigured, hasConnection bool) domain.Publisher {
	if metaConfigured && hasConnection {
		return real
	}
	return mock
}
