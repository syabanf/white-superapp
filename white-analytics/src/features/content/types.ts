/**
 * JSON shapes of white.content.v1 as grpc-gateway emits them: camelCase,
 * int64 as strings, enums by name. `optional` proto fields are omitted
 * when unset, and repeated fields may be missing when empty.
 */

export type Platform = "INSTAGRAM" | "FACEBOOK" | "TIKTOK";
export type PlanStatus = "PLAN_DRAFT" | "PLAN_IN_REVIEW" | "PLAN_APPROVED";
export type ContentFormat = "FEED" | "CAROUSEL" | "REEL" | "STORY";
export type ItemStatus =
  | "IDEA"
  | "COPYWRITING"
  | "DESIGN"
  | "REVIEW"
  | "REVISION"
  | "APPROVED"
  | "SCHEDULED"
  | "PUBLISHED"
  | "FAILED";
export type DesignStatus =
  | "DESIGN_TODO"
  | "DESIGN_IN_PROGRESS"
  | "DESIGN_IN_REVIEW"
  | "DESIGN_CHANGES_REQUESTED"
  | "DESIGN_APPROVED";
export type ReviewDecision = "COMMENT" | "REQUEST_CHANGES" | "APPROVE";
export type AssetKind = "LOGO" | "BRAND_GUIDELINE" | "FONT" | "PHOTO" | "TEMPLATE" | "OTHER";

export type PlatformFrequency = { platform: Platform; postsPerWeek: number };

export type PlanBrief = {
  goals: string;
  pillars?: string[];
  frequency?: PlatformFrequency[];
  tone: string;
  notes: string;
};

export type Plan = {
  id: string;
  clientId: string;
  month: string;
  title: string;
  status: PlanStatus;
  brief?: PlanBrief;
  itemCount: number;
  publishedCount: number;
  aiModel?: string;
  generatedAt?: string;
  submittedBy?: string;
  submittedAt?: string;
  approvedBy?: string;
  approvedAt?: string;
  reviewNote?: string;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
};

export type PlanComment = { id: string; userId?: string; userName: string; body: string; createdAt: string };

export type PlanDetail = { plan: Plan; items?: Item[]; comments?: PlanComment[] };

export type Item = {
  id: string;
  planId: string;
  clientId: string;
  plannedDate: string;
  publishAt?: string;
  platform: Platform;
  format: ContentFormat;
  pillar: string;
  title: string;
  hook: string;
  caption: string;
  hashtags?: string[];
  visualBrief: string;
  copywriterId?: string;
  designerId?: string;
  copyDue?: string;
  designDue?: string;
  reviewDue?: string;
  status: ItemStatus;
  revisionRound: number;
  captionVersion: number;
  publishingPostId?: string;
  publishedAt?: string;
  permalink?: string;
  publishError?: string;
  hasFinalDesign: boolean;
  approvedBy?: string;
  approvedAt?: string;
  createdBy?: string;
  updatedBy?: string;
  createdAt: string;
  updatedAt: string;
};

export type CaptionRevision = {
  id: string;
  version: number;
  caption: string;
  hook: string;
  hashtags?: string[];
  note: string;
  createdBy?: string;
  createdByName: string;
  createdAt: string;
};

export type ItemComment = {
  id: string;
  userId?: string;
  userName: string;
  body: string;
  /** COMMENT, CHANGES_REQUESTED or APPROVED. */
  kind: string;
  round: number;
  createdAt: string;
};

export type ItemActivity = {
  id: string;
  userId?: string;
  userName: string;
  type: string;
  fromStatus?: string;
  toStatus?: string;
  note?: string;
  createdAt: string;
};

export type ItemDetail = {
  item: Item;
  revisions?: CaptionRevision[];
  comments?: ItemComment[];
  activities?: ItemActivity[];
  designTasks?: DesignTask[];
  allowedMoves?: ItemStatus[];
};

/** Planning fields of CreateItem / UpdateItem. */
export type ItemInput = {
  plannedDate: string;
  publishAt?: string;
  platform: Platform;
  format: ContentFormat;
  pillar: string;
  title: string;
  visualBrief: string;
  copywriterId: string;
  designerId: string;
  copyDue: string;
  designDue: string;
  reviewDue: string;
};

export type SizePreset = { key: string; label: string; width: number; height: number; format: ContentFormat };

export type DesignTask = {
  id: string;
  itemId: string;
  clientId: string;
  title: string;
  brief: string;
  references?: string[];
  sizePreset: string;
  width: number;
  height: number;
  designerId?: string;
  dueDate?: string;
  status: DesignStatus;
  currentVersion: number;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
};

export type DesignFile = {
  id: string;
  taskId: string;
  version: number;
  sortOrder: number;
  fileName: string;
  mimeType: string;
  sizeBytes: string;
  uploadedBy?: string;
  createdAt: string;
};

export type DesignFeedback = {
  id: string;
  fileId?: string;
  version: number;
  userId?: string;
  userName: string;
  decision: ReviewDecision;
  body: string;
  createdAt: string;
};

export type DesignTaskDetail = { task: DesignTask; files?: DesignFile[]; feedback?: DesignFeedback[] };

export type DesignTaskInput = {
  title: string;
  brief: string;
  references: string[];
  sizePreset: string;
  designerId: string;
  dueDate: string;
};

export type AssetFolder = {
  id: string;
  clientId: string;
  parentId?: string;
  name: string;
  assetCount: number;
  createdAt: string;
};

export type AssetVersion = {
  id: string;
  version: number;
  fileName: string;
  mimeType: string;
  sizeBytes: string;
  note: string;
  uploadedBy?: string;
  createdAt: string;
};

export type Asset = {
  id: string;
  clientId: string;
  folderId?: string;
  kind: AssetKind;
  name: string;
  description: string;
  tags?: string[];
  currentVersion: number;
  current?: AssetVersion;
  createdBy?: string;
  updatedBy?: string;
  createdAt: string;
  updatedAt: string;
};

export type AssetDetail = { asset: Asset; versions?: AssetVersion[] };

export type ReportBreakdown = { key: string; planned: number; published: number };

export type ReportPost = {
  itemId: string;
  title: string;
  platform: Platform;
  format: ContentFormat;
  plannedDate: string;
  publishedAt?: string;
  permalink?: string;
  hasMetrics: boolean;
  likes: string;
  comments: string;
  shares: string;
  saves: string;
  reach: string;
  views: string;
  engagements: string;
};

export type ContentReport = {
  id: string;
  clientId: string;
  month: string;
  planned: number;
  published: number;
  failed: number;
  scheduled: number;
  inProgress: number;
  deadlinesMet: number;
  deadlinesTotal: number;
  onTimeRate: number;
  revisionsTotal: number;
  revisionAvg: number;
  captionVersions: number;
  perPlatform?: ReportBreakdown[];
  perFormat?: ReportBreakdown[];
  engagementAvailable: boolean;
  totalEngagements: string;
  totalReach: string;
  posts?: ReportPost[];
  top?: ReportPost[];
  bottom?: ReportPost[];
  summary: string;
  aiModel: string;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
};

/** A person who can be assigned on this client (admins plus members). */
export type Person = { id: string; name: string };
