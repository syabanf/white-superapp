package app

import (
	"context"
	"strconv"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Decks is the pitch deck automation: slide text drafted by OpenRouter
// (or the mock) over a fixed structure, written as PPTX and versioned.
type Decks struct{ *Deps }

func NewDecks(d *Deps) *Decks { return &Decks{d} }

func (d *Decks) List(ctx context.Context, dealID, leadID string) ([]domain.PitchDeck, error) {
	return d.Store.ListPitchDecks(ctx, dealID, leadID)
}

func (d *Decks) Get(ctx context.Context, id string) (*domain.PitchDeck, error) {
	deck, err := d.Store.GetPitchDeck(ctx, id)
	if err != nil {
		return nil, err
	}
	if deck == nil {
		return nil, notFound(msgDeckNotFound)
	}
	return deck, nil
}

// File returns the PPTX bytes and a download name.
func (d *Decks) File(ctx context.Context, id string) (string, []byte, error) {
	deck, err := d.Get(ctx, id)
	if err != nil {
		return "", nil, err
	}
	return fileName("Pitch-Deck-v"+strconv.Itoa(deck.Version), deck.Title, "pptx"), deck.File, nil
}

// input gathers the deal or lead data a deck is written from.
func (d *Decks) input(ctx context.Context, dealID, leadID string, opts domain.DeckOptions) (domain.DeckInput, error) {
	in := domain.DeckInput{Options: opts}
	if dealID != "" {
		deal, err := d.Store.GetDeal(ctx, dealID)
		if err != nil {
			return in, err
		}
		if deal == nil {
			return in, notFound(msgDealNotFound)
		}
		in.ClientName, in.PicName, in.Notes, in.ServiceLines = deal.ClientName(), deal.PicName, deref(deal.Notes), deal.ServiceLines
		if deal.AccountID != nil {
			if acc, err := d.Store.GetAccount(ctx, *deal.AccountID); err == nil && acc != nil {
				in.Industry = deref(acc.Industry)
			}
		}
		if deal.LeadID != nil {
			if lead, err := d.Store.GetLead(ctx, *deal.LeadID); err == nil && lead != nil && lead.Notes != nil {
				in.Notes = domain.JSTrim(in.Notes + "\n" + *lead.Notes)
			}
		}
		if in.Quotation, err = d.Store.LatestQuotation(ctx, dealID); err != nil {
			return in, err
		}
		names, err := d.teamNames(ctx, deal)
		if err != nil {
			return in, err
		}
		in.TeamNames = names
	} else {
		lead, err := d.Store.GetLead(ctx, leadID)
		if err != nil {
			return in, err
		}
		if lead == nil {
			return in, notFound(msgLeadNotFound)
		}
		in.ClientName, in.PicName, in.Notes = lead.OrgName, lead.PicName, deref(lead.Notes)
		in.ServiceLines = domain.DetectServiceLines(in.Notes)
	}
	if len(in.ServiceLines) == 0 {
		in.ServiceLines = []string{domain.LineSocialMedia}
	}
	if opts.IncludeCaseStudies {
		all, err := d.Store.ListCaseStudies(ctx, true)
		if err != nil {
			return in, err
		}
		in.CaseStudies = pickCaseStudies(all, in.ServiceLines, 3)
	}
	return in, nil
}

// pickCaseStudies prefers studies that share a service line.
func pickCaseStudies(all []domain.CaseStudy, lines []string, n int) []domain.CaseStudy {
	var match, rest []domain.CaseStudy
	for _, cs := range all {
		shared := false
		for _, l := range cs.ServiceLines {
			shared = shared || domain.Contains(lines, l)
		}
		if shared {
			match = append(match, cs)
		} else {
			rest = append(rest, cs)
		}
	}
	out := append(match, rest...)
	return out[:min(n, len(out))]
}

func (d *Decks) teamNames(ctx context.Context, deal *domain.Deal) ([]string, error) {
	if d.Directory == nil {
		return nil, nil
	}
	members, err := d.Store.ListDealMembers(ctx, deal.ID)
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	if deal.OwnerUserID != nil {
		wanted[*deal.OwnerUserID] = true
	}
	for _, m := range members {
		wanted[m.UserID] = true
	}
	users, err := d.Directory.ListUsers(ctx)
	if err != nil {
		d.warn("list users failed", "err", err)
		return nil, nil
	}
	var names []string
	for _, u := range users {
		if wanted[u.ID] {
			names = append(names, u.Name)
		}
	}
	return names, nil
}

// Generate writes a new deck version for a deal or a lead.
func (d *Decks) Generate(ctx context.Context, actor Actor, dealID, leadID string, opts domain.DeckOptions) (*domain.PitchDeck, error) {
	if (dealID == "") == (leadID == "") {
		return nil, invalid("Pilih salah satu: deal atau lead")
	}
	if opts.Tone != "" && !domain.Contains([]string{"profesional", "santai", "premium"}, opts.Tone) {
		return nil, invalid("Nada tidak valid")
	}
	in, err := d.input(ctx, dealID, leadID, opts)
	if err != nil {
		return nil, err
	}
	res, err := d.AI.Complete(ctx, AIRequest{Messages: []AIMessage{{Role: "system", Content: domain.DeckSystemPrompt(opts.Tone)}, {Role: "user", Content: domain.DeckUserMessage(in)}}, JSON: true, Temperature: 0.5, MaxTokens: 2000})
	if err != nil {
		return nil, err
	}
	copyText, model := domain.MockDeckCopy(in), MockModel
	if !res.Mock {
		if c, ok := domain.ParseDeckCopy(res.Content); ok {
			copyText, model = c, res.Model
		}
	}
	slides := domain.BuildSlides(in, copyText)
	title := "Proposal " + in.ClientName
	file, err := d.Renderer.PitchDeck(title, slides)
	if err != nil {
		return nil, err
	}
	version, err := d.Store.NextDeckVersion(ctx, dealID, leadID)
	if err != nil {
		return nil, err
	}
	deck := domain.PitchDeck{ID: ids.New(), DealID: domain.TrimmedOrNil(dealID), LeadID: domain.TrimmedOrNil(leadID), Title: title, Version: version, Options: opts, Slides: slides, File: file, SizeBytes: int64(len(file)), Model: model, CreatedBy: actor.ref(), CreatedAt: d.now()}
	if err := d.Store.InsertPitchDeck(ctx, deck); err != nil {
		return nil, err
	}
	return &deck, nil
}

func (d *Decks) ListCaseStudies(ctx context.Context) ([]domain.CaseStudy, error) {
	return d.Store.ListCaseStudies(ctx, false)
}

func (d *Decks) SaveCaseStudy(ctx context.Context, cs domain.CaseStudy) (*domain.CaseStudy, error) {
	errs := map[string]string{}
	if domain.JSTrim(cs.Title) == "" {
		errs["title"] = "Judul wajib diisi"
	}
	if domain.JSTrim(cs.ClientName) == "" {
		errs["clientName"] = "Nama klien wajib diisi"
	}
	for _, l := range cs.ServiceLines {
		if !domain.Contains(domain.ServiceLines, l) {
			errs["serviceLines"] = "Layanan tidak dikenal: " + l
		}
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if cs.ID == "" {
		cs.ID = ids.New()
	}
	return &cs, d.Store.SaveCaseStudy(ctx, cs)
}

func (d *Decks) DeleteCaseStudy(ctx context.Context, id string) error {
	return d.Store.DeleteCaseStudy(ctx, id)
}
