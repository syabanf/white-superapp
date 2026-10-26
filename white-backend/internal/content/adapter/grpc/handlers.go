package grpcapi

import (
	"context"

	contentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/content/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Services struct {
	Plans      *app.Plans
	Items      *app.Items
	Scheduling *app.Scheduling
	Designs    *app.Designs
	Assets     *app.Assets
	Reports    *app.Reports
	Dashboard  *app.Dashboard
}

func Register(s *grpc.Server, svc Services) {
	contentv1.RegisterPlanServiceServer(s, &planServer{svc: svc})
	contentv1.RegisterItemServiceServer(s, &itemServer{svc: svc})
	contentv1.RegisterDesignServiceServer(s, &designServer{svc: svc})
	contentv1.RegisterAssetServiceServer(s, &assetServer{svc: svc})
	contentv1.RegisterReportServiceServer(s, &reportServer{svc: svc})
}

// caller is the signed-in user id; every content RPC is client-scoped and
// the use case checks identity client access.
func caller(ctx context.Context) (string, error) {
	p, err := authn.Require(ctx)
	return p.UserID, err
}

// ─── PlanService ─────────────────────────────────────────────

type planServer struct {
	contentv1.UnimplementedPlanServiceServer
	svc Services
}

func planOut(d *app.PlanDetail, err error) (*contentv1.PlanDetail, error) {
	if err != nil {
		return nil, err
	}
	return planDetailPB(d), nil
}

func (s *planServer) ListPlans(ctx context.Context, req *contentv1.ClientRequest) (*contentv1.ListPlansResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	plans, err := s.svc.Plans.List(ctx, uid, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &contentv1.ListPlansResponse{}
	for _, p := range plans {
		out.Plans = append(out.Plans, planPB(p))
	}
	return out, nil
}

func (s *planServer) CreatePlan(ctx context.Context, req *contentv1.CreatePlanRequest) (*contentv1.PlanDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return planOut(s.svc.Plans.Create(ctx, uid, req.ClientId, req.Month, req.Title, briefFrom(req.Brief)))
}

func (s *planServer) GetPlan(ctx context.Context, req *contentv1.PlanRef) (*contentv1.PlanDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return planOut(s.svc.Plans.Get(ctx, uid, req.ClientId, req.PlanId))
}

func (s *planServer) UpdatePlan(ctx context.Context, req *contentv1.UpdatePlanRequest) (*contentv1.PlanDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	var brief *domain.Brief
	if req.Brief != nil {
		b := briefFrom(req.Brief)
		brief = &b
	}
	return planOut(s.svc.Plans.Update(ctx, uid, req.ClientId, req.PlanId, req.Title, brief))
}

func (s *planServer) DeletePlan(ctx context.Context, req *contentv1.PlanRef) (*emptypb.Empty, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Plans.Delete(ctx, uid, req.ClientId, req.PlanId)
}

func (s *planServer) GeneratePlanDraft(ctx context.Context, req *contentv1.GeneratePlanDraftRequest) (*contentv1.PlanDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return planOut(s.svc.Plans.Generate(ctx, uid, req.ClientId, req.PlanId, req.Replace))
}

func (s *planServer) SubmitPlan(ctx context.Context, req *contentv1.PlanRef) (*contentv1.PlanDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return planOut(s.svc.Plans.Submit(ctx, uid, req.ClientId, req.PlanId))
}

func (s *planServer) ApprovePlan(ctx context.Context, req *contentv1.PlanNoteRequest) (*contentv1.PlanDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return planOut(s.svc.Plans.Approve(ctx, uid, req.ClientId, req.PlanId, req.Note))
}

func (s *planServer) RequestPlanChanges(ctx context.Context, req *contentv1.PlanNoteRequest) (*contentv1.PlanDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return planOut(s.svc.Plans.RequestChanges(ctx, uid, req.ClientId, req.PlanId, req.Note))
}

func (s *planServer) AddPlanComment(ctx context.Context, req *contentv1.AddPlanCommentRequest) (*contentv1.PlanComment, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Plans.Comment(ctx, uid, req.ClientId, req.PlanId, req.Body)
	if err != nil {
		return nil, err
	}
	return planCommentPB(*c, s.svc.Plans.Identity.UserName(ctx, uid)), nil
}

// ─── ItemService ─────────────────────────────────────────────

type itemServer struct {
	contentv1.UnimplementedItemServiceServer
	svc Services
}

func itemOut(d *app.ItemDetail, err error) (*contentv1.ItemDetail, error) {
	if err != nil {
		return nil, err
	}
	return itemDetailPB(d), nil
}

func (s *itemServer) ListItems(ctx context.Context, req *contentv1.ListItemsRequest) (*contentv1.ListItemsResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	f := domain.ItemFilter{PlanID: req.PlanId, Status: itemStatusFrom(req.Status), AssigneeID: req.AssigneeId}
	if req.Month != "" {
		m, ok := domain.ParseMonth(req.Month)
		if !ok {
			return nil, monthErr()
		}
		f.Month = m
	}
	items, err := s.svc.Items.List(ctx, uid, req.ClientId, f)
	if err != nil {
		return nil, err
	}
	out := &contentv1.ListItemsResponse{}
	for _, it := range items {
		out.Items = append(out.Items, itemPB(it))
	}
	return out, nil
}

func (s *itemServer) CreateItem(ctx context.Context, req *contentv1.CreateItemRequest) (*contentv1.ItemDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return itemOut(s.svc.Items.Create(ctx, uid, req.ClientId, req.PlanId, itemInputFrom(req.Item), app.CaptionInput{Hook: req.Hook, Caption: req.Caption, Hashtags: req.Hashtags}))
}

func (s *itemServer) GetItem(ctx context.Context, req *contentv1.ItemRef) (*contentv1.ItemDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return itemOut(s.svc.Items.Get(ctx, uid, req.ClientId, req.ItemId))
}

func (s *itemServer) UpdateItem(ctx context.Context, req *contentv1.UpdateItemRequest) (*contentv1.ItemDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return itemOut(s.svc.Items.Update(ctx, uid, req.ClientId, req.ItemId, itemInputFrom(req.Item)))
}

func (s *itemServer) DeleteItem(ctx context.Context, req *contentv1.ItemRef) (*emptypb.Empty, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Items.Delete(ctx, uid, req.ClientId, req.ItemId)
}

func (s *itemServer) MoveItem(ctx context.Context, req *contentv1.MoveItemRequest) (*contentv1.ItemDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return itemOut(s.svc.Items.Move(ctx, uid, req.ClientId, req.ItemId, itemStatusFrom(req.To), req.Note))
}

func (s *itemServer) ReviseCaption(ctx context.Context, req *contentv1.ReviseCaptionRequest) (*contentv1.ItemDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return itemOut(s.svc.Items.Revise(ctx, uid, req.ClientId, req.ItemId, app.CaptionInput{Hook: req.Hook, Caption: req.Caption, Hashtags: req.Hashtags, Note: req.Note}))
}

func (s *itemServer) AddItemComment(ctx context.Context, req *contentv1.AddItemCommentRequest) (*contentv1.ItemComment, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Items.Comment(ctx, uid, req.ClientId, req.ItemId, req.Body)
	if err != nil {
		return nil, err
	}
	return itemCommentPB(*c, s.svc.Items.Identity.UserName(ctx, uid)), nil
}

func (s *itemServer) ScheduleItem(ctx context.Context, req *contentv1.ScheduleItemRequest) (*contentv1.ItemDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.svc.Scheduling.Schedule(ctx, uid, req.ClientId, req.ItemId, timePtr(req.PublishAt)); err != nil {
		return nil, err
	}
	return itemOut(s.svc.Items.Get(ctx, uid, req.ClientId, req.ItemId))
}

func (s *itemServer) SyncItemStatus(ctx context.Context, req *contentv1.ItemRef) (*contentv1.ItemDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Scheduling.Sync(ctx, uid, req.ClientId, req.ItemId); err != nil {
		return nil, err
	}
	return itemOut(s.svc.Items.Get(ctx, uid, req.ClientId, req.ItemId))
}

// ─── DesignService ───────────────────────────────────────────

type designServer struct {
	contentv1.UnimplementedDesignServiceServer
	svc Services
}

func designOut(d *app.DesignDetail, err error) (*contentv1.DesignTaskDetail, error) {
	if err != nil {
		return nil, err
	}
	return designDetailPB(d), nil
}

func (s *designServer) ListSizePresets(ctx context.Context, _ *emptypb.Empty) (*contentv1.ListSizePresetsResponse, error) {
	if _, err := caller(ctx); err != nil {
		return nil, err
	}
	out := &contentv1.ListSizePresetsResponse{}
	for _, p := range domain.SizePresets {
		out.Presets = append(out.Presets, &contentv1.SizePreset{Key: p.Key, Label: p.Label, Width: p.Width, Height: p.Height, Format: formatPB[p.Format]})
	}
	return out, nil
}

func (s *designServer) ListDesignTasks(ctx context.Context, req *contentv1.ListDesignTasksRequest) (*contentv1.ListDesignTasksResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := s.svc.Designs.List(ctx, uid, req.ClientId, req.ItemId, req.DesignerId, designStatusFrom(req.Status))
	if err != nil {
		return nil, err
	}
	out := &contentv1.ListDesignTasksResponse{}
	for _, t := range tasks {
		out.Tasks = append(out.Tasks, taskPB(t))
	}
	return out, nil
}

func (s *designServer) CreateDesignTask(ctx context.Context, req *contentv1.CreateDesignTaskRequest) (*contentv1.DesignTaskDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return designOut(s.svc.Designs.Create(ctx, uid, req.ClientId, req.ItemId, designInputFrom(req.Task)))
}

func (s *designServer) GetDesignTask(ctx context.Context, req *contentv1.DesignTaskRef) (*contentv1.DesignTaskDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return designOut(s.svc.Designs.Get(ctx, uid, req.ClientId, req.TaskId))
}

func (s *designServer) UpdateDesignTask(ctx context.Context, req *contentv1.UpdateDesignTaskRequest) (*contentv1.DesignTaskDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return designOut(s.svc.Designs.Update(ctx, uid, req.ClientId, req.TaskId, designInputFrom(req.Task), designStatusFrom(req.Status)))
}

func (s *designServer) UploadDesignFiles(ctx context.Context, req *contentv1.UploadDesignFilesRequest) (*contentv1.DesignTaskDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return designOut(s.svc.Designs.Upload(ctx, uid, req.ClientId, req.TaskId, uploadsFrom(req.Files), req.Note))
}

func (s *designServer) ReviewDesign(ctx context.Context, req *contentv1.ReviewDesignRequest) (*contentv1.DesignTaskDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return designOut(s.svc.Designs.Review(ctx, uid, req.ClientId, req.TaskId, decisionFrom(req.Decision), req.Body, req.FileId))
}

func (s *designServer) DownloadDesignFile(ctx context.Context, req *contentv1.DesignFileRef) (*contentv1.FileBytes, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	f, err := s.svc.Designs.Download(ctx, uid, req.ClientId, req.FileId)
	if err != nil {
		return nil, err
	}
	return fileBytesPB(f), nil
}

// ─── AssetService ────────────────────────────────────────────

type assetServer struct {
	contentv1.UnimplementedAssetServiceServer
	svc Services
}

func assetOut(d *app.AssetDetail, err error) (*contentv1.AssetDetail, error) {
	if err != nil {
		return nil, err
	}
	return assetDetailPB(d), nil
}

func folderOut(f *domain.AssetFolder, err error) (*contentv1.AssetFolder, error) {
	if err != nil {
		return nil, err
	}
	return folderPB(*f), nil
}

func (s *assetServer) ListAssetFolders(ctx context.Context, req *contentv1.ClientRequest) (*contentv1.ListAssetFoldersResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	folders, err := s.svc.Assets.ListFolders(ctx, uid, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &contentv1.ListAssetFoldersResponse{}
	for _, f := range folders {
		out.Folders = append(out.Folders, folderPB(f))
	}
	return out, nil
}

func (s *assetServer) CreateAssetFolder(ctx context.Context, req *contentv1.CreateAssetFolderRequest) (*contentv1.AssetFolder, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return folderOut(s.svc.Assets.CreateFolder(ctx, uid, req.ClientId, req.Name, req.ParentId))
}

func (s *assetServer) RenameAssetFolder(ctx context.Context, req *contentv1.RenameAssetFolderRequest) (*contentv1.AssetFolder, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return folderOut(s.svc.Assets.RenameFolder(ctx, uid, req.ClientId, req.FolderId, req.Name))
}

func (s *assetServer) DeleteAssetFolder(ctx context.Context, req *contentv1.AssetFolderRef) (*emptypb.Empty, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Assets.DeleteFolder(ctx, uid, req.ClientId, req.FolderId)
}

func (s *assetServer) SearchAssets(ctx context.Context, req *contentv1.SearchAssetsRequest) (*contentv1.SearchAssetsResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	assets, err := s.svc.Assets.Search(ctx, uid, req.ClientId, domain.AssetFilter{Query: req.Query, Kind: assetKindFrom(req.Kind), FolderID: req.FolderId, Tag: req.Tag})
	if err != nil {
		return nil, err
	}
	out := &contentv1.SearchAssetsResponse{}
	for _, a := range assets {
		out.Assets = append(out.Assets, assetPB(a))
	}
	return out, nil
}

func (s *assetServer) CreateAsset(ctx context.Context, req *contentv1.CreateAssetRequest) (*contentv1.AssetDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	in := app.AssetInput{Name: req.Name, Kind: assetKindFrom(req.Kind), Description: req.Description, Tags: req.Tags, FolderID: req.FolderId}
	return assetOut(s.svc.Assets.Create(ctx, uid, req.ClientId, in, app.UploadFile{Filename: req.File.GetFilename(), Content: req.File.GetContent()}))
}

func (s *assetServer) GetAsset(ctx context.Context, req *contentv1.AssetRef) (*contentv1.AssetDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return assetOut(s.svc.Assets.Get(ctx, uid, req.ClientId, req.AssetId))
}

func (s *assetServer) UpdateAsset(ctx context.Context, req *contentv1.UpdateAssetRequest) (*contentv1.AssetDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	in := app.AssetInput{Name: req.Name, Kind: assetKindFrom(req.Kind), Description: req.Description, Tags: req.Tags, FolderID: req.FolderId}
	return assetOut(s.svc.Assets.Update(ctx, uid, req.ClientId, req.AssetId, in))
}

func (s *assetServer) UploadAssetVersion(ctx context.Context, req *contentv1.UploadAssetVersionRequest) (*contentv1.AssetDetail, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return assetOut(s.svc.Assets.UploadVersion(ctx, uid, req.ClientId, req.AssetId, app.UploadFile{Filename: req.File.GetFilename(), Content: req.File.GetContent()}, req.Note))
}

func (s *assetServer) DeleteAsset(ctx context.Context, req *contentv1.AssetRef) (*emptypb.Empty, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Assets.Delete(ctx, uid, req.ClientId, req.AssetId)
}

func (s *assetServer) DownloadAsset(ctx context.Context, req *contentv1.DownloadAssetRequest) (*contentv1.FileBytes, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	f, err := s.svc.Assets.Download(ctx, uid, req.ClientId, req.AssetId, req.Version)
	if err != nil {
		return nil, err
	}
	return fileBytesPB(f), nil
}

// ─── ReportService ───────────────────────────────────────────

type reportServer struct {
	contentv1.UnimplementedReportServiceServer
	svc Services
}

func (s *reportServer) ListReports(ctx context.Context, req *contentv1.ClientRequest) (*contentv1.ListReportsResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	reports, err := s.svc.Reports.List(ctx, uid, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &contentv1.ListReportsResponse{}
	for i := range reports {
		out.Reports = append(out.Reports, reportPB(&reports[i]))
	}
	return out, nil
}

func (s *reportServer) GenerateReport(ctx context.Context, req *contentv1.GenerateReportRequest) (*contentv1.ContentReport, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Reports.Generate(ctx, uid, req.ClientId, req.Month)
	if err != nil {
		return nil, err
	}
	return reportPB(r), nil
}

func (s *reportServer) GetReport(ctx context.Context, req *contentv1.ReportRef) (*contentv1.ContentReport, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Reports.Get(ctx, uid, req.ClientId, req.ReportId)
	if err != nil {
		return nil, err
	}
	return reportPB(r), nil
}

func (s *reportServer) DownloadReportPdf(ctx context.Context, req *contentv1.ReportRef) (*contentv1.FileBytes, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	f, err := s.svc.Reports.PDF(ctx, uid, req.ClientId, req.ReportId)
	if err != nil {
		return nil, err
	}
	return fileBytesPB(f), nil
}
