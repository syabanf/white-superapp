package app

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Assets is the per-client brand asset library: folders, tags, search
// and versioned files.
type Assets struct{ Deps }

func NewAssets(d Deps) *Assets { return &Assets{d} }

var assetKinds = []domain.AssetKind{domain.AssetLogo, domain.AssetBrandGuideline, domain.AssetFont, domain.AssetPhoto, domain.AssetTemplate, domain.AssetOther}

type AssetInput struct {
	Name        string
	Kind        domain.AssetKind
	Description string
	Tags        []string
	FolderID    string
}

type AssetDetail struct {
	Asset    domain.Asset
	Versions []domain.AssetVersion
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range tags {
		t = strings.ToLower(trim(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func (s *Assets) validate(ctx context.Context, clientID string, in *AssetInput) error {
	v := validate.New()
	in.Name = trim(in.Name)
	if n := utf8.RuneCountInString(in.Name); n < 1 || n > 160 {
		v.Add("name", msgAssetName)
	}
	known := false
	for _, k := range assetKinds {
		known = known || k == in.Kind
	}
	if !known {
		v.Add("kind", msgAssetKind)
	}
	v.MaxLen("description", in.Description, 2000, "Deskripsi terlalu panjang (maksimal 2.000 karakter).")
	in.Tags = normalizeTags(in.Tags)
	if len(in.Tags) > 20 {
		v.Add("tags", "Maksimal 20 tag.")
	}
	if v.HasErrors() {
		return v.Err()
	}
	in.Description = trim(in.Description)
	if in.FolderID = trim(in.FolderID); in.FolderID != "" {
		if _, err := ensureFound(s.Store.GetFolder(ctx, clientID, in.FolderID))(msgFolderNotFound); err != nil {
			return err
		}
	}
	return nil
}

func folderName(name string) (string, error) {
	name = trim(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > 80 {
		return "", apperr.Field("name", msgFolderName)
	}
	return name, nil
}

func uniqueFolder(err error) error {
	var u *domain.ErrUnique
	if errors.As(err, &u) {
		return apperr.Field("name", msgFolderExists)
	}
	return err
}

func (s *Assets) ListFolders(ctx context.Context, userID, clientID string) ([]domain.AssetFolder, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return s.Store.ListFolders(ctx, clientID)
}

func (s *Assets) CreateFolder(ctx context.Context, userID, clientID, name, parentID string) (*domain.AssetFolder, error) {
	name, err := folderName(name)
	if err != nil {
		return nil, err
	}
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	parent := optID(parentID)
	if parent != nil {
		if _, err := ensureFound(s.Store.GetFolder(ctx, clientID, *parent))(msgFolderNotFound); err != nil {
			return nil, err
		}
	}
	f := domain.AssetFolder{ID: ids.New(), ClientID: clientID, ParentID: parent, Name: name, CreatedBy: &userID}
	if err := s.Store.InsertFolder(ctx, f); err != nil {
		return nil, uniqueFolder(err)
	}
	return s.Store.GetFolder(ctx, clientID, f.ID)
}

func (s *Assets) RenameFolder(ctx context.Context, userID, clientID, folderID, name string) (*domain.AssetFolder, error) {
	name, err := folderName(name)
	if err != nil {
		return nil, err
	}
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	f, err := ensureFound(s.Store.GetFolder(ctx, clientID, folderID))(msgFolderNotFound)
	if err != nil {
		return nil, err
	}
	if err := s.Store.RenameFolder(ctx, f.ID, name); err != nil {
		return nil, uniqueFolder(err)
	}
	return s.Store.GetFolder(ctx, clientID, f.ID)
}

func (s *Assets) DeleteFolder(ctx context.Context, userID, clientID, folderID string) error {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return err
	}
	f, err := ensureFound(s.Store.GetFolder(ctx, clientID, folderID))(msgFolderNotFound)
	if err != nil {
		return err
	}
	ok, err := s.Store.DeleteFolder(ctx, f.ID)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.New(apperr.Validation, msgFolderNotEmpty)
	}
	return nil
}

func (s *Assets) Search(ctx context.Context, userID, clientID string, f domain.AssetFilter) ([]domain.Asset, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	f.Query, f.Tag = trim(f.Query), strings.ToLower(trim(f.Tag))
	return s.Store.SearchAssets(ctx, clientID, f)
}

func (s *Assets) Create(ctx context.Context, userID, clientID string, in AssetInput, file UploadFile) (*AssetDetail, error) {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	if err := s.validate(ctx, clientID, &in); err != nil {
		return nil, err
	}
	stored, err := storeUploads(ctx, s.Files, clientID, []UploadFile{file})
	if err != nil {
		return nil, err
	}
	a := domain.Asset{ID: ids.New(), ClientID: clientID, FolderID: optID(in.FolderID), Kind: in.Kind, Name: in.Name, Description: in.Description, Tags: in.Tags, CurrentVersion: 1, CreatedBy: &userID}
	f := stored[0]
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertAsset(ctx, a); err != nil {
			return err
		}
		return tx.InsertAssetVersion(ctx, domain.AssetVersion{ID: ids.New(), AssetID: a.ID, Version: 1, FileKey: f.key, FileName: f.name, MimeType: f.mime, SizeBytes: f.size, UploadedBy: &userID})
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, clientID, a.ID)
}

func (s *Assets) Get(ctx context.Context, userID, clientID, assetID string) (*AssetDetail, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	a, err := ensureFound(s.Store.GetAsset(ctx, clientID, assetID))(msgAssetNotFound)
	if err != nil {
		return nil, err
	}
	versions, err := s.Store.ListAssetVersions(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		if versions[i].Version == a.CurrentVersion {
			a.Current = &versions[i]
		}
	}
	return &AssetDetail{Asset: *a, Versions: versions}, nil
}

func (s *Assets) Update(ctx context.Context, userID, clientID, assetID string, in AssetInput) (*AssetDetail, error) {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	a, err := ensureFound(s.Store.GetAsset(ctx, clientID, assetID))(msgAssetNotFound)
	if err != nil {
		return nil, err
	}
	if err := s.validate(ctx, clientID, &in); err != nil {
		return nil, err
	}
	a.Name, a.Kind, a.Description, a.Tags, a.FolderID, a.UpdatedBy = in.Name, in.Kind, in.Description, in.Tags, optID(in.FolderID), &userID
	if err := s.Store.UpdateAsset(ctx, *a); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, clientID, assetID)
}

// UploadVersion makes the file the current version; history is kept.
func (s *Assets) UploadVersion(ctx context.Context, userID, clientID, assetID string, file UploadFile, note string) (*AssetDetail, error) {
	note = trim(note)
	if utf8.RuneCountInString(note) > 1000 {
		return nil, apperr.Field("note", msgNoteRequired)
	}
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	a, err := ensureFound(s.Store.GetAsset(ctx, clientID, assetID))(msgAssetNotFound)
	if err != nil {
		return nil, err
	}
	stored, err := storeUploads(ctx, s.Files, clientID, []UploadFile{file})
	if err != nil {
		return nil, err
	}
	f := stored[0]
	a.CurrentVersion++
	a.UpdatedBy = &userID
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertAssetVersion(ctx, domain.AssetVersion{ID: ids.New(), AssetID: a.ID, Version: a.CurrentVersion, FileKey: f.key, FileName: f.name, MimeType: f.mime, SizeBytes: f.size, Note: note, UploadedBy: &userID}); err != nil {
			return err
		}
		return tx.UpdateAsset(ctx, *a)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, clientID, assetID)
}

func (s *Assets) Delete(ctx context.Context, userID, clientID, assetID string) error {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return err
	}
	a, err := ensureFound(s.Store.GetAsset(ctx, clientID, assetID))(msgAssetNotFound)
	if err != nil {
		return err
	}
	return s.Store.SoftDeleteAsset(ctx, a.ID, userID)
}

// Download returns one version's file; version 0 is the current one.
func (s *Assets) Download(ctx context.Context, userID, clientID, assetID string, version int32) (*File, error) {
	d, err := s.Get(ctx, userID, clientID, assetID)
	if err != nil {
		return nil, err
	}
	if version == 0 {
		version = d.Asset.CurrentVersion
	}
	for _, v := range d.Versions {
		if v.Version == version {
			return readFile(ctx, s.Files, v.FileKey, v.FileName, v.MimeType)
		}
	}
	return nil, apperr.New(apperr.NotFound, msgVersionNotFound)
}
