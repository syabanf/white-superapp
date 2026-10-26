package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The pure rules of lib/crm/public-forms.ts (crm/publicforms/rules.go).

// FormField is one field of a public form definition.
type FormField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // text | textarea | email | phone | number | date | select | checkbox
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Help     string   `json:"help,omitempty"`
}

var FormFieldTypes = []string{"text", "textarea", "email", "phone", "number", "date", "select", "checkbox"}

var FieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,39}$`)
var FormSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,60}$`)

// ValidateFormFields checks a form definition.
func ValidateFormFields(fields []FormField) string {
	if len(fields) == 0 {
		return "Form minimal punya 1 field"
	}
	seen := map[string]bool{}
	for _, f := range fields {
		switch {
		case !FieldKeyRe.MatchString(f.Key):
			return "Key field tidak valid: " + f.Key
		case seen[f.Key]:
			return "Key field duplikat: " + f.Key
		case JSTrim(f.Label) == "":
			return "Label field wajib diisi"
		case !Contains(FormFieldTypes, f.Type):
			return "Tipe field tidak valid: " + f.Type
		}
		seen[f.Key] = true
	}
	return ""
}

// leadMappedKeys are LEAD_MAPPED_KEYS: fields stored on lead columns; the
// rest go to the custom jsonb.
var leadMappedKeys = map[string]bool{
	"org_name": true, "pic_name": true, "pic_phone": true, "pic_email": true, "city": true, "notes": true, "org_type": true,
}

// Attribution is the UTM and page attribution of a submission.
type Attribution struct {
	UtmSource   *string `json:"utm_source,omitempty"`
	UtmMedium   *string `json:"utm_medium,omitempty"`
	UtmCampaign *string `json:"utm_campaign,omitempty"`
	UtmContent  *string `json:"utm_content,omitempty"`
	UtmTerm     *string `json:"utm_term,omitempty"`
	LandingPage *string `json:"landing_page,omitempty"`
	Referrer    *string `json:"referrer,omitempty"`
}

func clean(v any, max int) *string {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	if s = SliceUTF16(JSTrim(s), max); s == "" {
		return nil
	}
	return &s
}

// ParseAttribution is parseAttribution.
func ParseAttribution(p map[string]any) Attribution {
	return Attribution{
		UtmSource: clean(p["utm_source"], 100), UtmMedium: clean(p["utm_medium"], 100),
		UtmCampaign: clean(p["utm_campaign"], 150), UtmContent: clean(p["utm_content"], 150),
		UtmTerm: clean(p["utm_term"], 150), LandingPage: clean(p["landing_page"], 500), Referrer: clean(p["referrer"], 500),
	}
}

var knownSources = map[string]string{
	"instagram": "instagram", "ig": "instagram", "facebook": "instagram", "meta": "instagram",
	"google": "google", "googleads": "google", "google_ads": "google", "adwords": "google", "sem": "google",
	"wa": "wa", "whatsapp": "wa",
	"linkedin": "linkedin",
	"referral": "referral", "refferal": "referral",
	"pameran": "pameran", "event": "pameran", "expo": "pameran",
	"website": "website", "web": "website", "organic": "website",
}

var notSourceChar = regexp.MustCompile(`[^a-z_]`)

// SourceFromAttribution is sourceFromAttribution: utm_source mapped to a
// known lead source, else the form default.
func SourceFromAttribution(a Attribution, fallback string) string {
	src := ""
	if a.UtmSource != nil {
		src = *a.UtmSource
	}
	if s, ok := knownSources[notSourceChar.ReplaceAllString(strings.ToLower(src), "")]; ok {
		return s
	}
	return ToLeadSource(fallback)
}

// ToLeadSource is toLeadSource: a valid lead source, or "lainnya".
func ToLeadSource(v string) string {
	if Contains(LeadSources, v) {
		return v
	}
	return "lainnya"
}

// FieldError is one field's validation message.
type FieldError struct {
	Key     string
	Message string
}

// Submission is SubmissionResult: Lead keeps only mapped keys.
type Submission struct {
	Errors []FieldError
	Lead   map[string]string
	Custom map[string]any
}

func (s Submission) OK() bool { return len(s.Errors) == 0 }

var (
	nonPhn = regexp.MustCompile(`[^0-9+]`)
	nonNum = regexp.MustCompile(`[^0-9.-]`)
	nonDig = regexp.MustCompile(`\D`)
)

// ValidateSubmission is validateSubmission: each defined field checked by
// type, unknown keys dropped, and a phone or email always required.
func ValidateSubmission(fields []FormField, payload map[string]any) Submission {
	out := Submission{Lead: map[string]string{}, Custom: map[string]any{}}
	for _, f := range fields {
		raw := payload[f.Key]
		if raw == nil || raw == "" || raw == false {
			if f.Required {
				out.Errors = append(out.Errors, FieldError{f.Key, f.Label + " wajib diisi"})
			}
			continue
		}
		var value any
		switch f.Type {
		case "email":
			s := SliceUTF16(JSTrim(ValueString(raw)), 150)
			if !IsEmail(s) {
				out.Errors = append(out.Errors, FieldError{f.Key, f.Label + " bukan email yang valid"})
				continue
			}
			value = s
		case "phone":
			digits := nonPhn.ReplaceAllString(ValueString(raw), "")
			if len(nonDig.ReplaceAllString(digits, "")) < 8 {
				out.Errors = append(out.Errors, FieldError{f.Key, f.Label + " terlalu pendek"})
				continue
			}
			value = SliceUTF16(digits, 20)
		case "number":
			digits := nonNum.ReplaceAllString(ValueString(raw), "")
			n, err := strconv.ParseFloat(digits, 64)
			if digits == "" || err != nil || math.IsInf(n, 0) {
				out.Errors = append(out.Errors, FieldError{f.Key, f.Label + " harus berupa angka"})
				continue
			}
			value = n
		case "date":
			s := JSTrim(ValueString(raw))
			if !IsValidCalendarDate(s) {
				out.Errors = append(out.Errors, FieldError{f.Key, f.Label + " harus berupa tanggal"})
				continue
			}
			value = s
		case "select":
			s := JSTrim(ValueString(raw))
			if len(f.Options) > 0 && !Contains(f.Options, s) {
				out.Errors = append(out.Errors, FieldError{f.Key, f.Label + " tidak ada di pilihan"})
				continue
			}
			value = s
		case "checkbox":
			value = raw == true || raw == "true" || raw == "on" || raw == "1" || raw == float64(1)
		case "textarea":
			value = SliceUTF16(JSTrim(ValueString(raw)), 2000)
		default:
			value = SliceUTF16(JSTrim(ValueString(raw)), 200)
		}
		if leadMappedKeys[f.Key] {
			out.Lead[f.Key] = ValueString(value)
		} else {
			out.Custom[f.Key] = value
		}
	}
	flagged := false
	for _, e := range out.Errors {
		if e.Key == "pic_phone" || e.Key == "pic_email" {
			flagged = true
		}
	}
	if out.Lead["pic_phone"] == "" && out.Lead["pic_email"] == "" && !flagged {
		out.Errors = append(out.Errors, FieldError{"pic_phone", "Isi nomor WhatsApp atau email agar kami bisa menghubungi Anda"})
	}
	return out
}

const (
	honeypotField  = "website_url"
	minFillSeconds = 3
)

// IsLikelyBot is isLikelyBot: a filled honeypot, or a form timestamp that
// is in the future, over a day old, or under 3 seconds old.
func IsLikelyBot(p map[string]any, now time.Time) (bool, string) {
	if trap, ok := p[honeypotField].(string); ok && JSTrim(trap) != "" {
		return true, "honeypot terisi"
	}
	started := ValueNumber(p["form_started_at"])
	if !math.IsNaN(started) && started > 0 {
		elapsed := (float64(now.UnixMilli()) - started) / 1000
		if elapsed < 0 || elapsed > 24*3600 {
			return true, "stempel waktu tidak wajar"
		}
		if elapsed < minFillSeconds {
			return true, "kiriman terlalu cepat"
		}
	}
	return false, ""
}

// LeadOrgName is leadOrgName.
func LeadOrgName(lead map[string]string) string {
	if s := JSTrim(lead["org_name"]); s != "" {
		return s
	}
	if s := JSTrim(lead["pic_name"]); s != "" {
		return s
	}
	return "Kiriman Form Publik"
}

// BuildLeadAlert is buildLeadAlert: the notification text to sales.
func BuildLeadAlert(formName string, lead map[string]string, a Attribution) string {
	pic := lead["pic_name"]
	if pic == "" {
		pic = "-"
	}
	if lead["pic_phone"] != "" {
		pic += " · " + lead["pic_phone"]
	}
	lines := []string{"Form: " + formName, "Instansi: " + LeadOrgName(lead), "PIC: " + pic}
	if lead["pic_email"] != "" {
		lines = append(lines, "Email: "+lead["pic_email"])
	}
	if lead["city"] != "" {
		lines = append(lines, "Kota: "+lead["city"])
	}
	if lead["notes"] != "" {
		lines = append(lines, "Kebutuhan: "+SliceUTF16(lead["notes"], 300))
	}
	var utm []string
	for _, v := range []*string{a.UtmSource, a.UtmMedium, a.UtmCampaign} {
		if v != nil {
			utm = append(utm, *v)
		}
	}
	if len(utm) > 0 {
		lines = append(lines, "Asal: "+strings.Join(utm, " / "))
	}
	return strings.Join(lines, "\n")
}

// HashIP stores the submitter's IP as a hash only.
func HashIP(ip string) string {
	sum := sha256.Sum256([]byte("white-form:" + ip))
	return hex.EncodeToString(sum[:])
}
