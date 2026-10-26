package app

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Designs is the design workflow on an item: tasks with a brief and size,
// versioned file uploads and reviewer feedback.
type Designs struct{ Deps }

func NewDesigns(d Deps) *Designs { return &Designs{d} }

type DesignInput struct {
	Title      string
	Brief      string
	References []string
	SizePreset string
	DesignerID string
	DueDate    string
}

type DesignDetail struct {
	Task     domain.DesignTask
	Files    []domain.DesignFile
	Feedback []domain.DesignFeedback
	Names    map[string]string
}

type UploadFile struct {
	Filename string
	Content  []byte
}

// File is a stored file read back for download.
type File struct {
	Name        string
	ContentType string
	Data        []byte
}

type storedUpload struct {
	key, name, mime string
	size            int64
}

// storeUploads validates every file (size, sniffed type) before writing any.
func storeUploads(ctx context.Context, files domain.Files, clientID string, uploads []UploadFile) ([]storedUpload, error) {
	type checked struct {
		f         UploadFile
		mime, ext string
	}
	var ok []checked
	for _, f := range uploads {
		if len(f.Content) == 0 {
			return nil, apperr.Field("files", msgNoFiles)
		}
		if len(f.Content) > domain.MaxUploadBytes {
			return nil, apperr.Field("files", msgFileTooLarge)
		}
		mime, ext := domain.SniffFile(f.Content)
		if mime == "" {
			return nil, apperr.Field("files", msgFileType)
		}
		ok = append(ok, checked{f, mime, ext})
	}
	out := make([]storedUpload, 0, len(ok))
	for _, c := range ok {
		obj, err := files.Put(ctx, clientID, c.ext, c.f.Content)
		if err != nil {
			return nil, err
		}
		out = append(out, storedUpload{key: obj.Key, name: domain.CleanFilename(c.f.Filename), mime: c.mime, size: obj.SizeBytes})
	}
	return out, nil
}

func readFile(ctx context.Context, files domain.Files, key, name, mime string) (*File, error) {
	data, err := files.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, apperr.New(apperr.NotFound, msgFileNotFound)
	}
	return &File{Name: name, ContentType: mime, Data: data}, nil
}

func applyDesignInput(t *domain.DesignTask, in DesignInput, format domain.Format) error {
	v := validate.New()
	v.MaxLen("title", in.Title, 160, msgTitleTooLong)
	v.MaxLen("brief", in.Brief, 4000, "Brief terlalu panjang (maksimal 4.000 karakter).")
	refs := []string{}
	for _, r := range in.References {
		if r = trim(r); r != "" {
			v.MaxLen("references", r, 500, "Referensi terlalu panjang (maksimal 500 karakter).")
			refs = append(refs, r)
		}
	}
	if len(refs) > 20 {
		v.Add("references", "Maksimal 20 referensi.")
	}
	preset := domain.DefaultSizePreset(format)
	if key := trim(in.SizePreset); key != "" {
		p, ok := domain.FindSizePreset(key)
		if !ok {
			v.Add("sizePreset", msgPresetInvalid)
		}
		preset = p
	}
	due := parseDay(v, "dueDate", in.DueDate)
	if v.HasErrors() {
		return v.Err()
	}
	if title := trim(in.Title); title != "" {
		t.Title = title
	}
	t.Brief, t.References = trim(in.Brief), refs
	t.SizePreset, t.Width, t.Height = preset.Key, preset.Width, preset.Height
	t.DesignerID, t.DueDate = optID(in.DesignerID), due
	return nil
}

func (s *Designs) Create(ctx context.Context, userID, clientID, itemID string, in DesignInput) (*DesignDetail, error) {
	a, err := s.manage(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	it, err := ensureFound(s.Store.GetItem(ctx, clientID, itemID))(msgItemNotFound)
	if err != nil {
		return nil, err
	}
	if domain.IsLocked(it.Status) {
		return nil, apperr.New(apperr.Validation, msgItemLocked)
	}
	t := domain.DesignTask{ID: ids.New(), ItemID: it.ID, ClientID: clientID, Title: "Desain: " + it.Title, Status: domain.DesignTodo, CreatedBy: &userID}
	if in.DesignerID == "" && it.DesignerID != nil {
		in.DesignerID = *it.DesignerID
	}
	if in.DueDate == "" && it.DesignDue != nil {
		in.DueDate = it.DesignDue.Format(time.DateOnly)
	}
	if in.Brief == "" {
		in.Brief = it.VisualBrief
	}
	if err := applyDesignInput(&t, in, it.Format); err != nil {
		return nil, err
	}
	if err := s.Store.InsertDesignTask(ctx, t); err != nil {
		return nil, err
	}
	if t.DesignerID != nil && *t.DesignerID != userID {
		s.notify(ctx, NotifyInput{UserIDs: []string{*t.DesignerID}, ClientID: clientID, Type: NotifDesignAssigned, Title: "Tugas desain baru: " + t.Title, Href: itemHref(a.Slug, it.ID)})
	}
	return s.Get(ctx, userID, clientID, t.ID)
}

func (s *Designs) List(ctx context.Context, userID, clientID, itemID, designerID string, status domain.DesignStatus) ([]domain.DesignTask, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return s.Store.ListDesignTasks(ctx, clientID, itemID, designerID, status)
}

func (s *Designs) Get(ctx context.Context, userID, clientID, taskID string) (*DesignDetail, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	t, err := ensureFound(s.Store.GetDesignTask(ctx, clientID, taskID))(msgTaskNotFound)
	if err != nil {
		return nil, err
	}
	d := &DesignDetail{Task: *t}
	if d.Files, err = s.Store.ListDesignFiles(ctx, t.ID); err != nil {
		return nil, err
	}
	if d.Feedback, err = s.Store.ListDesignFeedback(ctx, t.ID); err != nil {
		return nil, err
	}
	people := make([]*string, 0, len(d.Feedback))
	for _, f := range d.Feedback {
		people = append(people, f.UserID)
	}
	d.Names = s.userNames(ctx, people...)
	return d, nil
}

func (s *Designs) loadForWrite(ctx context.Context, userID, clientID, taskID string) (*domain.DesignTask, *Access, error) {
	a, err := s.manage(ctx, userID, clientID)
	if err != nil {
		return nil, nil, err
	}
	t, err := ensureFound(s.Store.GetDesignTask(ctx, clientID, taskID))(msgTaskNotFound)
	if err != nil {
		return nil, nil, err
	}
	return t, a, nil
}

// Update edits the task. By hand the status only moves between TODO and
// IN_PROGRESS; review states come from Upload and Review.
func (s *Designs) Update(ctx context.Context, userID, clientID, taskID string, in DesignInput, status domain.DesignStatus) (*DesignDetail, error) {
	t, _, err := s.loadForWrite(ctx, userID, clientID, taskID)
	if err != nil {
		return nil, err
	}
	it, err := ensureFound(s.Store.GetItem(ctx, clientID, t.ItemID))(msgItemNotFound)
	if err != nil {
		return nil, err
	}
	switch status {
	case "":
	case domain.DesignTodo, domain.DesignInProgress:
		t.Status = status
	default:
		return nil, apperr.Field("status", msgDesignStatus)
	}
	if err := applyDesignInput(t, in, it.Format); err != nil {
		return nil, err
	}
	t.UpdatedBy = &userID
	if err := s.Store.UpdateDesignTask(ctx, *t); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, clientID, taskID)
}

// Upload stores the files as the next version and puts the task in review.
func (s *Designs) Upload(ctx context.Context, userID, clientID, taskID string, files []UploadFile, note string) (*DesignDetail, error) {
	if len(files) == 0 {
		return nil, apperr.Field("files", msgNoFiles)
	}
	if len(files) > 10 {
		return nil, apperr.Field("files", msgTooManyFiles)
	}
	note = trim(note)
	if utf8.RuneCountInString(note) > 2000 {
		return nil, apperr.Field("note", msgCommentRequired)
	}
	t, a, err := s.loadForWrite(ctx, userID, clientID, taskID)
	if err != nil {
		return nil, err
	}
	stored, err := storeUploads(ctx, s.Files, clientID, files)
	if err != nil {
		return nil, err
	}
	t.CurrentVersion++
	t.Status, t.UpdatedBy = domain.DesignInReview, &userID
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		for i, f := range stored {
			if err := tx.InsertDesignFile(ctx, domain.DesignFile{ID: ids.New(), TaskID: t.ID, Version: t.CurrentVersion, SortOrder: int32(i), FileKey: f.key, FileName: f.name, MimeType: f.mime, SizeBytes: f.size, UploadedBy: &userID}); err != nil {
				return err
			}
		}
		if note != "" {
			if err := tx.InsertDesignFeedback(ctx, domain.DesignFeedback{ID: ids.New(), TaskID: t.ID, Version: t.CurrentVersion, UserID: &userID, Decision: domain.DecisionComment, Body: note}); err != nil {
				return err
			}
		}
		return tx.UpdateDesignTask(ctx, *t)
	})
	if err != nil {
		return nil, err
	}
	s.notify(ctx, NotifyInput{ClientID: clientID, Audience: AudienceClientApprovers, ExceptUserID: userID, Type: NotifDesignReview, Title: "Desain menunggu review: " + t.Title, Href: itemHref(a.Slug, t.ItemID)})
	return s.Get(ctx, userID, clientID, taskID)
}

// Review records feedback. Comments are open to every member; approving
// or requesting changes is for a reviewer other than the designer (ADMIN
// excepted).
func (s *Designs) Review(ctx context.Context, userID, clientID, taskID, decision, body, fileID string) (*DesignDetail, error) {
	switch decision {
	case domain.DecisionComment, domain.DecisionChangesRequested, domain.DecisionApproved:
	default:
		return nil, apperr.Field("decision", "Keputusan review tidak valid.")
	}
	body = trim(body)
	if utf8.RuneCountInString(body) > 2000 || (decision != domain.DecisionApproved && body == "") {
		return nil, apperr.Field("body", msgCommentRequired)
	}
	a, err := s.access(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	t, err := ensureFound(s.Store.GetDesignTask(ctx, clientID, taskID))(msgTaskNotFound)
	if err != nil {
		return nil, err
	}
	if decision != domain.DecisionComment {
		if !a.canWrite() {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		if t.CurrentVersion == 0 {
			return nil, apperr.New(apperr.Validation, msgReviewNoVersion)
		}
		if t.DesignerID != nil && *t.DesignerID == userID && !domain.CanApproveOwn(a.Role) {
			return nil, apperr.New(apperr.Unauthorized, msgSelfApprove)
		}
	}
	fb := domain.DesignFeedback{ID: ids.New(), TaskID: t.ID, FileID: optID(fileID), Version: t.CurrentVersion, UserID: &userID, Decision: decision, Body: body}
	t.Status = domain.DesignStatusAfterReview(t.Status, decision)
	t.UpdatedBy = &userID
	now := s.now()
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertDesignFeedback(ctx, fb); err != nil {
			return err
		}
		if decision == domain.DecisionComment {
			return nil
		}
		if err := tx.UpdateDesignTask(ctx, *t); err != nil {
			return err
		}
		if decision != domain.DecisionApproved {
			return nil
		}
		it, err := tx.GetItem(ctx, clientID, t.ItemID)
		if err != nil || it == nil {
			return err
		}
		if it.DesignDoneAt == nil {
			it.DesignDoneAt = &now
			if err := tx.UpdateItem(ctx, *it); err != nil {
				return err
			}
		}
		return tx.InsertActivity(ctx, domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: &userID, Type: domain.ActDesignDone, Note: optID(t.Title)})
	})
	if err != nil {
		return nil, err
	}
	if t.DesignerID != nil && *t.DesignerID != userID {
		title := "Feedback desain: " + t.Title
		switch decision {
		case domain.DecisionApproved:
			title = "Desain disetujui: " + t.Title
		case domain.DecisionChangesRequested:
			title = "Desain perlu revisi: " + t.Title
		}
		s.notify(ctx, NotifyInput{UserIDs: []string{*t.DesignerID}, ClientID: clientID, Type: NotifDesignReview, Title: title, Body: body, Href: itemHref(a.Slug, t.ItemID)})
	}
	return s.Get(ctx, userID, clientID, taskID)
}

func (s *Designs) Download(ctx context.Context, userID, clientID, fileID string) (*File, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	f, err := ensureFound(s.Store.GetDesignFile(ctx, clientID, fileID))(msgFileNotFound)
	if err != nil {
		return nil, err
	}
	return readFile(ctx, s.Files, f.FileKey, f.FileName, f.MimeType)
}
