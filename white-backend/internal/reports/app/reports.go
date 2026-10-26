package app

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Messages copied from features/reports/strings.ts.
const (
	msgTitleRequired = "Judul laporan wajib diisi."
	msgNoModules     = "Pilih minimal satu modul."
	msgInvalidRange  = "Periode laporan tidak valid."
)

// pdfTopN covers the PDF's 7 top posts and 8 top queries.
const pdfTopN = 8

type Reports struct{ Deps }

func New(d Deps) *Reports { return &Reports{Deps: d} }

// ReportRow is listReports: the report plus its creator's name.
type ReportRow struct {
	domain.Report
	CreatedByName *string
}

func parseRange(from, to string) (dates.Range, error) {
	f, errF := dates.Parse(from)
	t, errT := dates.Parse(to)
	if errF != nil || errT != nil || f.After(t) {
		return dates.Range{}, apperr.New(apperr.Validation, msgInvalidRange)
	}
	return dates.Range{From: f, To: t}, nil
}

func (r *Reports) List(ctx context.Context, userID, clientID string) ([]ReportRow, error) {
	if _, err := r.Identity.Check(ctx, userID, clientID); err != nil {
		return nil, err
	}
	rows, err := r.Store.ListReports(ctx, clientID)
	if err != nil {
		return nil, err
	}
	names := map[string]*string{}
	out := make([]ReportRow, 0, len(rows))
	for _, row := range rows {
		item := ReportRow{Report: row}
		if id := row.CreatedByID; id != nil {
			name, seen := names[*id]
			if !seen {
				n, exists, err := r.Identity.UserName(ctx, *id)
				if err != nil {
					return nil, err
				}
				if exists {
					name = &n
				}
				names[*id] = name
			}
			item.CreatedByName = name
		}
		out = append(out, item)
	}
	return out, nil
}

type CreateInput struct {
	Title     string
	Modules   []string
	Compare   bool
	IncludeAI bool
	Language  string
	From      string
	To        string
}

// Create is createReport: any member may create. With IncludeAI the
// OVERVIEW insight is generated and cached first; its failure aborts.
func (r *Reports) Create(ctx context.Context, userID, clientID string, in CreateInput) (*ReportRow, error) {
	acc, err := r.Identity.Check(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if in.Language == "" {
		in.Language = "id"
	}
	v := validate.New()
	if title == "" {
		v.Add("title", msgTitleRequired)
	} else if utf8.RuneCountInString(title) > 160 {
		v.Add("title", "Too big: expected string to have <=160 characters")
	}
	if len(in.Modules) == 0 {
		v.Add("modules", msgNoModules)
	}
	for _, m := range in.Modules {
		if !slices.Contains(domain.ReportModules, m) {
			v.Add("modules", `Invalid option: expected one of "SOCIAL"|"SEO"|"ADS"`)
			break
		}
	}
	v.OneOf("language", in.Language, []string{"id", "en"}, `Invalid option: expected one of "id"|"en"`)
	if err := v.Err(); err != nil {
		return nil, err
	}
	rg, err := parseRange(in.From, in.To)
	if err != nil {
		return nil, err
	}

	var aiSummary *string
	if in.IncludeAI {
		insight, err := r.generate(ctx, acc.Client, domain.ModuleOverview, rg, in.Language, in.Modules)
		if err != nil {
			return nil, err
		}
		aiSummary = &insight.Content
	}

	// created_by_id only when the user row still exists (sessions can
	// outlive a re-seeded database).
	var createdBy, createdByName *string
	if name, exists, err := r.Identity.UserName(ctx, userID); err != nil {
		return nil, err
	} else if exists {
		createdBy, createdByName = &userID, &name
	}
	report, err := r.Store.InsertReport(ctx, domain.Report{
		ID: ids.New(), ClientID: clientID, Title: title, DateFrom: rg.From, DateTo: rg.To, Compare: in.Compare,
		Modules: in.Modules, Language: in.Language, AiSummary: aiSummary, CreatedByID: createdBy, CreatedAt: r.Clock.Now(),
	})
	if err != nil {
		return nil, err
	}
	return &ReportRow{Report: *report, CreatedByName: createdByName}, nil
}

// Delete is deleteReport: MANAGER or ADMIN only.
func (r *Reports) Delete(ctx context.Context, userID, clientID, reportID string) error {
	acc, err := r.Identity.Check(ctx, userID, clientID)
	if err != nil {
		return err
	}
	if !acc.CanManage {
		return apperr.New(apperr.Unauthorized, "")
	}
	ok, err := r.Store.DeleteReport(ctx, clientID, reportID)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.New(apperr.NotFound, "")
	}
	return nil
}

func validModule(module string) error {
	if !domain.IsInsightModule(module) {
		return apperr.New(apperr.Validation, "")
	}
	return nil
}

// GenerateInsight is generateInsight: any member, cached in ai_insights.
func (r *Reports) GenerateInsight(ctx context.Context, userID, clientID, module, from, to, language string) (*domain.Insight, error) {
	acc, err := r.Identity.Check(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	if err := validModule(module); err != nil {
		return nil, err
	}
	rg, err := parseRange(from, to)
	if err != nil {
		return nil, err
	}
	return r.generate(ctx, acc.Client, module, rg, domain.Lang(language), nil)
}

func (r *Reports) generate(ctx context.Context, client ClientInfo, module string, rg dates.Range, lang string, selection []string) (*domain.Insight, error) {
	prev := dates.Previous(rg)
	data, err := r.Analytics.ReportData(ctx, client.ID, rg, prev, domain.InsightModules(module, selection), domain.InsightTopN(module))
	if err != nil {
		return nil, err
	}
	content, model, err := r.Insights.Generate(ctx, domain.InsightInput{
		Module: module, Language: lang, ClientName: client.Name,
		Range: domain.ToRangeInfo(rg, prev), Data: domain.BuildInsightPayload(module, *data, client.Currency),
	})
	if err != nil {
		return nil, err
	}
	return r.Store.InsertInsight(ctx, domain.Insight{
		ID: ids.New(), ClientID: client.ID, Module: module, DateFrom: rg.From, DateTo: rg.To,
		Language: lang, Model: model, Content: content, CreatedAt: r.Clock.Now(),
	})
}

// LatestInsight is getLatestInsight for the exact range (any language).
func (r *Reports) LatestInsight(ctx context.Context, userID, clientID, module, from, to string) (*domain.Insight, error) {
	if _, err := r.Identity.Check(ctx, userID, clientID); err != nil {
		return nil, err
	}
	if err := validModule(module); err != nil {
		return nil, err
	}
	rg, err := parseRange(from, to)
	if err != nil {
		return nil, err
	}
	return r.Store.LatestInsight(ctx, clientID, module, rg.From, rg.To)
}

// Sections is getReportSections for the builder preview.
func (r *Reports) Sections(ctx context.Context, userID, clientID, from, to string, compare bool, modules []string, lang string) (domain.Sections, error) {
	if _, err := r.Identity.Check(ctx, userID, clientID); err != nil {
		return domain.Sections{}, err
	}
	rg, err := parseRange(from, to)
	if err != nil {
		return domain.Sections{}, err
	}
	return r.sections(ctx, clientID, rg, dates.Previous(rg), compare, modules, domain.Lang(lang))
}

func (r *Reports) sections(ctx context.Context, clientID string, rg, prev dates.Range, compare bool, modules []string, lang string) (domain.Sections, error) {
	var in domain.SectionInputs
	var err error
	if slices.Contains(modules, domain.ModuleSocial) {
		if in.Publishing, err = r.Publishing.Summary(ctx, clientID, rg); err != nil {
			return domain.Sections{}, err
		}
	}
	if slices.Contains(modules, domain.ModuleSeo) {
		pid, err := r.Analytics.SuitePropertyID(ctx, clientID)
		if err != nil {
			return domain.Sections{}, err
		}
		if pid != "" {
			if in.Rank, err = r.SeoSuite.Rank(ctx, pid, rg, prev); err != nil {
				return domain.Sections{}, err
			}
			if in.Backlinks, err = r.SeoSuite.Backlinks(ctx, pid, rg); err != nil {
				return domain.Sections{}, err
			}
		}
	}
	return domain.BuildSections(in, compare, domain.LabelsFor(lang)), nil
}

type pdfRequest struct {
	client    ClientInfo
	title     string
	rg        dates.Range
	compare   bool
	modules   []string
	language  string
	aiSummary string
}

// renderPdf recomputes every number live; nothing is snapshotted.
func (r *Reports) renderPdf(ctx context.Context, p pdfRequest) (string, []byte, error) {
	prev := dates.Previous(p.rg)
	data := &domain.ReportData{}
	if len(p.modules) > 0 {
		var err error
		if data, err = r.Analytics.ReportData(ctx, p.client.ID, p.rg, prev, p.modules, pdfTopN); err != nil {
			return "", nil, err
		}
	}
	lang := domain.Lang(p.language)
	sections, err := r.sections(ctx, p.client.ID, p.rg, prev, p.compare, p.modules, lang)
	if err != nil {
		return "", nil, err
	}
	pdfData := domain.BuildPdfData(domain.PdfArgs{
		ClientName: p.client.Name, Currency: p.client.Currency, Title: p.title, Current: p.rg, Previous: prev,
		Compare: p.compare, Modules: p.modules, Language: lang, AiSummary: p.aiSummary, Now: r.Clock.Now(),
	}, *data, sections)
	pdf, err := r.Render(pdfData)
	if err != nil {
		return "", nil, err
	}
	return domain.PdfFilename(p.client.Slug, p.rg.From, p.rg.To), pdf, nil
}

// RenderReportPdf is GET /api/reports/{id}/pdf: any member of the report's client.
func (r *Reports) RenderReportPdf(ctx context.Context, userID, reportID string) (string, []byte, error) {
	report, err := r.Store.GetReport(ctx, reportID)
	if err != nil {
		return "", nil, err
	}
	if report == nil {
		return "", nil, apperr.New(apperr.NotFound, "")
	}
	acc, err := r.Identity.Check(ctx, userID, report.ClientID)
	if err != nil {
		return "", nil, err
	}
	summary := ""
	if report.AiSummary != nil {
		summary = *report.AiSummary
	}
	return r.renderPdf(ctx, pdfRequest{
		client: acc.Client, title: report.Title, rg: dates.Range{From: report.DateFrom, To: report.DateTo},
		compare: report.Compare, modules: report.Modules, language: report.Language, aiSummary: summary,
	})
}

// share resolves the public context and the preset range.
func (r *Reports) share(ctx context.Context, slug, unlockToken, preset string) (*ShareContext, dates.Resolved, error) {
	sc, err := r.Identity.ShareContext(ctx, slug, unlockToken)
	if err != nil {
		return nil, dates.Resolved{}, err
	}
	preset = domain.SharePreset(preset)
	cur := dates.ForPreset(preset, r.Clock.Now())
	return sc, dates.Resolved{Current: cur, Previous: dates.Previous(cur), Preset: preset, Compare: true}, nil
}

type SharePdf struct {
	Status   ShareStatus
	Filename string
	Pdf      []byte
}

// SharePdf is GET /api/share/{slug}/pdf: compare on, the shared modules,
// Indonesian, and the latest OVERVIEW insight of that exact range.
func (r *Reports) SharePdf(ctx context.Context, slug, unlockToken, preset string) (*SharePdf, error) {
	sc, rr, err := r.share(ctx, slug, unlockToken, preset)
	if err != nil {
		return nil, err
	}
	if sc.Status != ShareOK {
		return &SharePdf{Status: sc.Status}, nil
	}
	summary := ""
	insight, err := r.Store.LatestInsight(ctx, sc.Client.ID, domain.ModuleOverview, rr.Current.From, rr.Current.To)
	if err != nil {
		return nil, err
	}
	if insight != nil {
		summary = insight.Content
	}
	name, pdf, err := r.renderPdf(ctx, pdfRequest{
		client: sc.Client, title: domain.ShareTitle(sc.Client.Name), rg: rr.Current, compare: true,
		modules: sc.Client.ShareModules, language: "id", aiSummary: summary,
	})
	if err != nil {
		return nil, err
	}
	return &SharePdf{Status: ShareOK, Filename: name, Pdf: pdf}, nil
}

type ShareDashboard struct {
	Status ShareStatus
	Client ClientInfo
	Range  dates.Resolved
	// Blocks of the shared modules that have data (see domain.ShareView).
	View domain.ReportData
}

// ShareDashboard is the public share page data.
func (r *Reports) ShareDashboard(ctx context.Context, slug, unlockToken, preset string) (*ShareDashboard, error) {
	sc, rr, err := r.share(ctx, slug, unlockToken, preset)
	if err != nil {
		return nil, err
	}
	out := &ShareDashboard{Status: sc.Status, Client: sc.Client, Range: rr}
	if sc.Status != ShareOK || len(sc.Client.ShareModules) == 0 {
		return out, nil
	}
	data, err := r.Analytics.ReportData(ctx, sc.Client.ID, rr.Current, rr.Previous, sc.Client.ShareModules, 10)
	if err != nil {
		return nil, err
	}
	out.View = domain.ShareView(*data, rr.Current.From)
	return out, nil
}

// PurgeClient handles client.deleted.
func (r *Reports) PurgeClient(ctx context.Context, clientID string) (int64, int64, error) {
	return r.Store.PurgeClient(ctx, clientID)
}

// ForgetUser handles user.deleted.
func (r *Reports) ForgetUser(ctx context.Context, userID string) (int64, error) {
	return r.Store.ClearCreator(ctx, userID)
}
