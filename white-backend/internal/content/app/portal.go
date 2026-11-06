package app

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// ShareGate checks a share slug and its unlock cookie through identity. It
// returns nil unless the page is enabled, protected by a PIN and unlocked.
type ShareGate interface {
	UnlockedClient(ctx context.Context, slug, unlockToken string) (*ClientInfo, error)
}

// Portal lets a client approve content in review, or send it back with a
// note, from the share page.
type Portal struct{ Deps }

func NewPortal(d Deps) *Portal { return &Portal{d} }

type PortalItem struct {
	Item  domain.Item
	Files []domain.DesignFile
}

const (
	msgPortalLocked = "Tautan tidak valid atau PIN belum dimasukkan"
	notifClientDone = "CONTENT_CLIENT_DECISION"
)

func (p *Portal) client(ctx context.Context, slug, token string) (*ClientInfo, error) {
	if p.Share == nil {
		return nil, apperr.New(apperr.NotFound, msgPortalLocked)
	}
	c, err := p.Share.UnlockedClient(ctx, slug, token)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.New(apperr.NotFound, msgPortalLocked)
	}
	return c, nil
}

// Review lists the items waiting for review with their final design files.
func (p *Portal) Review(ctx context.Context, slug, token string) ([]PortalItem, error) {
	c, err := p.client(ctx, slug, token)
	if err != nil {
		return nil, err
	}
	items, err := p.Store.ListItems(ctx, c.ID, domain.ItemFilter{Status: domain.ItemReview})
	if err != nil {
		return nil, err
	}
	out := make([]PortalItem, 0, len(items))
	for _, it := range items {
		files, err := p.Store.FinalDesignFiles(ctx, it.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, PortalItem{Item: it, Files: files})
	}
	return out, nil
}

// Decide approves the item or sends it to revision. The client has no user
// id, so the activity and comment carry the signer's name in the note.
func (p *Portal) Decide(ctx context.Context, slug, token, itemID string, approve bool, signer, note string) error {
	c, err := p.client(ctx, slug, token)
	if err != nil {
		return err
	}
	signer, note = strings.TrimSpace(signer), strings.TrimSpace(note)
	switch {
	case signer == "":
		return apperr.Field("signer_name", "Nama Anda wajib diisi")
	case !approve && note == "":
		return apperr.Field("note", "Tulis apa yang perlu diubah")
	case utf8.RuneCountInString(note) > 1000:
		return apperr.Field("note", "Catatan maksimal 1000 karakter")
	}
	it, err := p.Store.GetItem(ctx, c.ID, itemID)
	if err != nil {
		return err
	}
	if it == nil || it.Status != domain.ItemReview {
		return apperr.New(apperr.Conflict, "Konten ini sudah tidak menunggu persetujuan")
	}
	to, kind, verb := domain.ItemRevision, domain.DecisionChangesRequested, "minta revisi"
	if approve {
		if it.Caption == "" && it.Format != domain.FormatStory {
			return apperr.New(apperr.Validation, msgCaptionRequired)
		}
		to, kind, verb = domain.ItemApproved, domain.DecisionApproved, "menyetujui"
	}
	now := p.now()
	domain.MoveStamps(it, to, "", now)
	it.ApprovedBy, it.UpdatedBy = nil, nil
	body := "Klien (" + signer + ") " + verb
	if note != "" {
		body += ": " + note
	}
	from, toS := string(domain.ItemReview), string(to)
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateItem(ctx, *it); err != nil {
			return err
		}
		if err := tx.InsertActivity(ctx, domain.ItemActivity{ID: ids.New(), ItemID: it.ID, Type: domain.ActMoved, FromStatus: &from, ToStatus: &toS, Note: &body}); err != nil {
			return err
		}
		return tx.InsertItemComment(ctx, domain.ItemComment{ID: ids.New(), ItemID: it.ID, Body: body, Kind: kind, Round: it.RevisionRound})
	})
	if err != nil {
		return err
	}
	var team []string
	for _, id := range []*string{it.CopywriterID, it.DesignerID, it.CreatedBy} {
		if id != nil {
			team = append(team, *id)
		}
	}
	p.notify(ctx, NotifyInput{UserIDs: team, ClientID: c.ID, Audience: AudienceClientApprovers, Type: notifClientDone,
		Title: c.Name + " " + verb + " \"" + it.Title + "\"", Body: note, Href: itemHref(c.Slug, it.ID)})
	return nil
}

// File returns a final design file of an item the client is reviewing.
func (p *Portal) File(ctx context.Context, slug, token, fileID string) (*domain.DesignFile, []byte, error) {
	c, err := p.client(ctx, slug, token)
	if err != nil {
		return nil, nil, err
	}
	f, err := p.Store.GetDesignFile(ctx, c.ID, fileID)
	if err != nil {
		return nil, nil, err
	}
	if f == nil {
		return nil, nil, apperr.New(apperr.NotFound, "File tidak ditemukan")
	}
	data, err := p.Files.Get(ctx, f.FileKey)
	return f, data, err
}
