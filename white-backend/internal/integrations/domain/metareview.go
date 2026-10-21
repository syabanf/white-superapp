package domain

import "encoding/json"

type MetaReviewStatus string

const (
	ReviewNotRequested MetaReviewStatus = "NOT_REQUESTED"
	ReviewSubmitted    MetaReviewStatus = "SUBMITTED"
	ReviewApproved     MetaReviewStatus = "APPROVED"
)

func (s MetaReviewStatus) Valid() bool {
	return s == ReviewNotRequested || s == ReviewSubmitted || s == ReviewApproved
}

type MetaPermission struct {
	Name string
	// Unlocks is the product area the permission enables: social, publish, ads, engagement.
	Unlocks  string
	Required bool
}

// MetaPermissions is the App Review checklist. The optional ones are a later phase.
var MetaPermissions = []MetaPermission{
	{Name: "pages_show_list", Unlocks: "social", Required: true},
	{Name: "pages_read_engagement", Unlocks: "social", Required: true},
	{Name: "instagram_basic", Unlocks: "social", Required: true},
	{Name: "instagram_manage_insights", Unlocks: "social", Required: true},
	{Name: "business_management", Unlocks: "ads", Required: true},
	{Name: "ads_read", Unlocks: "ads", Required: true},
	{Name: "pages_manage_posts", Unlocks: "publish", Required: true},
	{Name: "instagram_content_publish", Unlocks: "publish", Required: true},
	{Name: "instagram_manage_comments", Unlocks: "engagement", Required: false},
	{Name: "pages_messaging", Unlocks: "engagement", Required: false},
}

type MetaReviewState struct {
	Permissions      map[string]MetaReviewStatus `json:"permissions"`
	AppLive          bool                        `json:"appLive"`
	BusinessVerified bool                        `json:"businessVerified"`
}

// ParseMetaReview reads the review state out of the META setting's meta
// JSON. Unknown statuses are dropped; booleans are coerced the JS way.
func ParseMetaReview(raw []byte) MetaReviewState {
	out := MetaReviewState{Permissions: map[string]MetaReviewStatus{}}
	if len(raw) == 0 {
		return out
	}
	var o struct {
		Permissions      map[string]any `json:"permissions"`
		AppLive          any            `json:"appLive"`
		BusinessVerified any            `json:"businessVerified"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		return out
	}
	for k, v := range o.Permissions {
		s, ok := v.(string)
		if ok && MetaReviewStatus(s).Valid() {
			out.Permissions[k] = MetaReviewStatus(s)
		}
	}
	out.AppLive = truthy(o.AppLive)
	out.BusinessVerified = truthy(o.BusinessVerified)
	return out
}

// truthy mirrors JavaScript Boolean(): false, 0, "", null are false.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x != ""
	}
	return true
}

func (s MetaReviewState) StatusOf(permission string) MetaReviewStatus {
	if v, ok := s.Permissions[permission]; ok {
		return v
	}
	return ReviewNotRequested
}

type ReviewProgress struct {
	Approved     int
	Total        int
	PublishReady bool
	AdsReady     bool
}

// Progress counts approved required permissions. An area is ready when
// every required permission in it is approved and the app is live.
func (s MetaReviewState) Progress() ReviewProgress {
	var p ReviewProgress
	for _, perm := range MetaPermissions {
		if !perm.Required {
			continue
		}
		p.Total++
		if s.StatusOf(perm.Name) == ReviewApproved {
			p.Approved++
		}
	}
	p.PublishReady = s.areaReady("publish")
	p.AdsReady = s.areaReady("ads")
	return p
}

func (s MetaReviewState) areaReady(area string) bool {
	for _, perm := range MetaPermissions {
		if perm.Unlocks == area && perm.Required && s.StatusOf(perm.Name) != ReviewApproved {
			return false
		}
	}
	return s.AppLive
}
