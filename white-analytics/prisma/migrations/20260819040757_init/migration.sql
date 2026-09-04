-- CreateEnum
CREATE TYPE "UserRole" AS ENUM ('ADMIN', 'MEMBER');

-- CreateEnum
CREATE TYPE "ClientMemberRole" AS ENUM ('MANAGER', 'VIEWER');

-- CreateEnum
CREATE TYPE "ConnectionProvider" AS ENUM ('META', 'GOOGLE');

-- CreateEnum
CREATE TYPE "SocialPlatform" AS ENUM ('INSTAGRAM', 'FACEBOOK', 'TIKTOK');

-- CreateEnum
CREATE TYPE "SeoDimension" AS ENUM ('QUERY', 'PAGE', 'COUNTRY', 'DEVICE');

-- CreateEnum
CREATE TYPE "AuditStrategy" AS ENUM ('MOBILE', 'DESKTOP');

-- CreateEnum
CREATE TYPE "IssueSeverity" AS ENUM ('ERROR', 'WARNING', 'NOTICE');

-- CreateEnum
CREATE TYPE "JobStatus" AS ENUM ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED');

-- CreateEnum
CREATE TYPE "ModuleKey" AS ENUM ('SOCIAL', 'SEO', 'ADS');

-- CreateTable
CREATE TABLE "User" (
    "id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "passwordHash" TEXT NOT NULL,
    "role" "UserRole" NOT NULL DEFAULT 'MEMBER',
    "isActive" BOOLEAN NOT NULL DEFAULT true,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Client" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "slug" TEXT NOT NULL,
    "description" TEXT NOT NULL DEFAULT '',
    "logoUrl" TEXT,
    "websiteUrl" TEXT,
    "industry" TEXT,
    "currency" TEXT NOT NULL DEFAULT 'IDR',
    "timezone" TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    "shareEnabled" BOOLEAN NOT NULL DEFAULT false,
    "sharePinHash" TEXT,
    "shareModules" "ModuleKey"[] DEFAULT ARRAY['SOCIAL', 'SEO', 'ADS']::"ModuleKey"[],
    "createdById" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Client_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "ClientMember" (
    "userId" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "role" "ClientMemberRole" NOT NULL DEFAULT 'VIEWER',
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "ClientMember_pkey" PRIMARY KEY ("userId","clientId")
);

-- CreateTable
CREATE TABLE "Connection" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "provider" "ConnectionProvider" NOT NULL,
    "accountId" TEXT NOT NULL,
    "displayName" TEXT NOT NULL,
    "accessTokenEnc" TEXT NOT NULL,
    "refreshTokenEnc" TEXT,
    "expiresAt" TIMESTAMP(3),
    "scopes" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "meta" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Connection_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SocialAccount" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "platform" "SocialPlatform" NOT NULL,
    "externalId" TEXT NOT NULL,
    "username" TEXT NOT NULL,
    "displayName" TEXT NOT NULL,
    "avatarUrl" TEXT,
    "biography" TEXT,
    "isCompetitor" BOOLEAN NOT NULL DEFAULT false,
    "connectionId" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "SocialAccount_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SocialSnapshot" (
    "id" TEXT NOT NULL,
    "socialAccountId" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "followers" INTEGER NOT NULL,
    "following" INTEGER,
    "mediaCount" INTEGER NOT NULL DEFAULT 0,
    "reach" INTEGER,
    "impressions" INTEGER,
    "profileViews" INTEGER,
    "websiteClicks" INTEGER,

    CONSTRAINT "SocialSnapshot_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SocialPost" (
    "id" TEXT NOT NULL,
    "socialAccountId" TEXT NOT NULL,
    "externalId" TEXT NOT NULL,
    "caption" TEXT NOT NULL DEFAULT '',
    "mediaType" TEXT NOT NULL DEFAULT 'IMAGE',
    "productType" TEXT,
    "permalink" TEXT,
    "mediaUrl" TEXT,
    "thumbnailUrl" TEXT,
    "publishedAt" TIMESTAMP(3) NOT NULL,
    "likes" INTEGER NOT NULL DEFAULT 0,
    "comments" INTEGER NOT NULL DEFAULT 0,
    "shares" INTEGER NOT NULL DEFAULT 0,
    "saves" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "reach" INTEGER NOT NULL DEFAULT 0,
    "impressions" INTEGER NOT NULL DEFAULT 0,
    "syncedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "SocialPost_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SeoProperty" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "siteUrl" TEXT NOT NULL,
    "ga4PropertyId" TEXT,
    "connectionId" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "SeoProperty_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SeoDailyMetric" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "clicks" INTEGER NOT NULL,
    "impressions" INTEGER NOT NULL,
    "ctr" DOUBLE PRECISION NOT NULL,
    "position" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "SeoDailyMetric_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SeoDimensionMetric" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "dimension" "SeoDimension" NOT NULL,
    "key" TEXT NOT NULL,
    "clicks" INTEGER NOT NULL,
    "impressions" INTEGER NOT NULL,
    "ctr" DOUBLE PRECISION NOT NULL,
    "position" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "SeoDimensionMetric_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Ga4DailyMetric" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "sessions" INTEGER NOT NULL,
    "organicSessions" INTEGER NOT NULL,
    "users" INTEGER NOT NULL,
    "engagedSessions" INTEGER NOT NULL,
    "conversions" INTEGER NOT NULL,
    "avgEngagementSec" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "Ga4DailyMetric_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SeoAudit" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "strategy" "AuditStrategy" NOT NULL,
    "runAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "performance" INTEGER NOT NULL,
    "seo" INTEGER NOT NULL,
    "accessibility" INTEGER NOT NULL,
    "bestPractices" INTEGER NOT NULL,
    "lcpMs" DOUBLE PRECISION,
    "inpMs" DOUBLE PRECISION,
    "cls" DOUBLE PRECISION,
    "fcpMs" DOUBLE PRECISION,
    "ttfbMs" DOUBLE PRECISION,
    "tbtMs" DOUBLE PRECISION,
    "speedIndexMs" DOUBLE PRECISION,
    "raw" JSONB,

    CONSTRAINT "SeoAudit_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SeoCrawl" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "startedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "finishedAt" TIMESTAMP(3),
    "status" "JobStatus" NOT NULL DEFAULT 'PENDING',
    "pagesCrawled" INTEGER NOT NULL DEFAULT 0,
    "maxPages" INTEGER NOT NULL DEFAULT 50,
    "error" TEXT,

    CONSTRAINT "SeoCrawl_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SeoCrawlPage" (
    "id" TEXT NOT NULL,
    "crawlId" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "statusCode" INTEGER NOT NULL,
    "title" TEXT,
    "metaDescription" TEXT,
    "h1Count" INTEGER NOT NULL DEFAULT 0,
    "wordCount" INTEGER NOT NULL DEFAULT 0,
    "canonical" TEXT,
    "indexable" BOOLEAN NOT NULL DEFAULT true,
    "imagesTotal" INTEGER NOT NULL DEFAULT 0,
    "imagesMissingAlt" INTEGER NOT NULL DEFAULT 0,
    "internalLinks" INTEGER NOT NULL DEFAULT 0,
    "externalLinks" INTEGER NOT NULL DEFAULT 0,
    "loadMs" INTEGER,
    "hasJsonLd" BOOLEAN NOT NULL DEFAULT false,
    "hasHreflang" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "SeoCrawlPage_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SeoIssue" (
    "id" TEXT NOT NULL,
    "crawlId" TEXT NOT NULL,
    "severity" "IssueSeverity" NOT NULL,
    "code" TEXT NOT NULL,
    "message" TEXT NOT NULL,
    "url" TEXT NOT NULL,

    CONSTRAINT "SeoIssue_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AdAccount" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "externalId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "currency" TEXT NOT NULL DEFAULT 'IDR',
    "connectionId" TEXT,
    "lastSyncedAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "AdAccount_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AdCampaign" (
    "id" TEXT NOT NULL,
    "adAccountId" TEXT NOT NULL,
    "externalId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "objective" TEXT NOT NULL DEFAULT 'OUTCOME_TRAFFIC',
    "status" TEXT NOT NULL DEFAULT 'ACTIVE',
    "dailyBudget" DOUBLE PRECISION,
    "startTime" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "AdCampaign_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AdSet" (
    "id" TEXT NOT NULL,
    "campaignId" TEXT NOT NULL,
    "externalId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'ACTIVE',
    "dailyBudget" DOUBLE PRECISION,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "AdSet_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Ad" (
    "id" TEXT NOT NULL,
    "adSetId" TEXT NOT NULL,
    "externalId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'ACTIVE',
    "thumbnailUrl" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Ad_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AdDailyInsight" (
    "id" TEXT NOT NULL,
    "adAccountId" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "campaignId" TEXT,
    "adSetId" TEXT,
    "adId" TEXT,
    "spend" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "impressions" INTEGER NOT NULL DEFAULT 0,
    "reach" INTEGER NOT NULL DEFAULT 0,
    "clicks" INTEGER NOT NULL DEFAULT 0,
    "linkClicks" INTEGER NOT NULL DEFAULT 0,
    "results" INTEGER NOT NULL DEFAULT 0,
    "resultType" TEXT,
    "purchaseValue" DOUBLE PRECISION,

    CONSTRAINT "AdDailyInsight_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AdDemographic" (
    "id" TEXT NOT NULL,
    "adAccountId" TEXT NOT NULL,
    "dateFrom" DATE NOT NULL,
    "dateTo" DATE NOT NULL,
    "campaignId" TEXT,
    "age" TEXT NOT NULL,
    "gender" TEXT NOT NULL,
    "spend" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "impressions" INTEGER NOT NULL DEFAULT 0,
    "reach" INTEGER NOT NULL DEFAULT 0,
    "results" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "AdDemographic_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Report" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "dateFrom" DATE NOT NULL,
    "dateTo" DATE NOT NULL,
    "compare" BOOLEAN NOT NULL DEFAULT true,
    "modules" "ModuleKey"[] DEFAULT ARRAY['SOCIAL', 'SEO', 'ADS']::"ModuleKey"[],
    "language" TEXT NOT NULL DEFAULT 'id',
    "aiSummary" TEXT,
    "pdfPath" TEXT,
    "createdById" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Report_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AiInsight" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "module" TEXT NOT NULL,
    "dateFrom" DATE NOT NULL,
    "dateTo" DATE NOT NULL,
    "language" TEXT NOT NULL DEFAULT 'id',
    "model" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "AiInsight_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SyncJob" (
    "id" TEXT NOT NULL,
    "clientId" TEXT,
    "kind" TEXT NOT NULL,
    "status" "JobStatus" NOT NULL DEFAULT 'PENDING',
    "startedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "finishedAt" TIMESTAMP(3),
    "message" TEXT,
    "error" TEXT,

    CONSTRAINT "SyncJob_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");

-- CreateIndex
CREATE UNIQUE INDEX "Client_slug_key" ON "Client"("slug");

-- CreateIndex
CREATE INDEX "Client_createdById_idx" ON "Client"("createdById");

-- CreateIndex
CREATE INDEX "ClientMember_clientId_idx" ON "ClientMember"("clientId");

-- CreateIndex
CREATE INDEX "Connection_clientId_idx" ON "Connection"("clientId");

-- CreateIndex
CREATE UNIQUE INDEX "Connection_clientId_provider_accountId_key" ON "Connection"("clientId", "provider", "accountId");

-- CreateIndex
CREATE INDEX "SocialAccount_clientId_platform_idx" ON "SocialAccount"("clientId", "platform");

-- CreateIndex
CREATE UNIQUE INDEX "SocialAccount_clientId_platform_externalId_key" ON "SocialAccount"("clientId", "platform", "externalId");

-- CreateIndex
CREATE INDEX "SocialSnapshot_socialAccountId_date_idx" ON "SocialSnapshot"("socialAccountId", "date");

-- CreateIndex
CREATE UNIQUE INDEX "SocialSnapshot_socialAccountId_date_key" ON "SocialSnapshot"("socialAccountId", "date");

-- CreateIndex
CREATE INDEX "SocialPost_socialAccountId_publishedAt_idx" ON "SocialPost"("socialAccountId", "publishedAt");

-- CreateIndex
CREATE UNIQUE INDEX "SocialPost_socialAccountId_externalId_key" ON "SocialPost"("socialAccountId", "externalId");

-- CreateIndex
CREATE INDEX "SeoProperty_clientId_idx" ON "SeoProperty"("clientId");

-- CreateIndex
CREATE UNIQUE INDEX "SeoProperty_clientId_siteUrl_key" ON "SeoProperty"("clientId", "siteUrl");

-- CreateIndex
CREATE INDEX "SeoDailyMetric_propertyId_date_idx" ON "SeoDailyMetric"("propertyId", "date");

-- CreateIndex
CREATE UNIQUE INDEX "SeoDailyMetric_propertyId_date_key" ON "SeoDailyMetric"("propertyId", "date");

-- CreateIndex
CREATE INDEX "SeoDimensionMetric_propertyId_dimension_date_idx" ON "SeoDimensionMetric"("propertyId", "dimension", "date");

-- CreateIndex
CREATE UNIQUE INDEX "SeoDimensionMetric_propertyId_date_dimension_key_key" ON "SeoDimensionMetric"("propertyId", "date", "dimension", "key");

-- CreateIndex
CREATE INDEX "Ga4DailyMetric_propertyId_date_idx" ON "Ga4DailyMetric"("propertyId", "date");

-- CreateIndex
CREATE UNIQUE INDEX "Ga4DailyMetric_propertyId_date_key" ON "Ga4DailyMetric"("propertyId", "date");

-- CreateIndex
CREATE INDEX "SeoAudit_propertyId_runAt_idx" ON "SeoAudit"("propertyId", "runAt");

-- CreateIndex
CREATE INDEX "SeoCrawl_propertyId_startedAt_idx" ON "SeoCrawl"("propertyId", "startedAt");

-- CreateIndex
CREATE INDEX "SeoCrawlPage_crawlId_idx" ON "SeoCrawlPage"("crawlId");

-- CreateIndex
CREATE INDEX "SeoIssue_crawlId_severity_idx" ON "SeoIssue"("crawlId", "severity");

-- CreateIndex
CREATE INDEX "AdAccount_clientId_idx" ON "AdAccount"("clientId");

-- CreateIndex
CREATE UNIQUE INDEX "AdAccount_clientId_externalId_key" ON "AdAccount"("clientId", "externalId");

-- CreateIndex
CREATE INDEX "AdCampaign_adAccountId_idx" ON "AdCampaign"("adAccountId");

-- CreateIndex
CREATE UNIQUE INDEX "AdCampaign_adAccountId_externalId_key" ON "AdCampaign"("adAccountId", "externalId");

-- CreateIndex
CREATE INDEX "AdSet_campaignId_idx" ON "AdSet"("campaignId");

-- CreateIndex
CREATE UNIQUE INDEX "AdSet_campaignId_externalId_key" ON "AdSet"("campaignId", "externalId");

-- CreateIndex
CREATE INDEX "Ad_adSetId_idx" ON "Ad"("adSetId");

-- CreateIndex
CREATE UNIQUE INDEX "Ad_adSetId_externalId_key" ON "Ad"("adSetId", "externalId");

-- CreateIndex
CREATE INDEX "AdDailyInsight_adAccountId_date_idx" ON "AdDailyInsight"("adAccountId", "date");

-- CreateIndex
CREATE INDEX "AdDailyInsight_campaignId_date_idx" ON "AdDailyInsight"("campaignId", "date");

-- CreateIndex
CREATE INDEX "AdDailyInsight_adId_date_idx" ON "AdDailyInsight"("adId", "date");

-- CreateIndex
CREATE UNIQUE INDEX "AdDailyInsight_adAccountId_date_campaignId_adSetId_adId_key" ON "AdDailyInsight"("adAccountId", "date", "campaignId", "adSetId", "adId");

-- CreateIndex
CREATE INDEX "AdDemographic_adAccountId_dateFrom_idx" ON "AdDemographic"("adAccountId", "dateFrom");

-- CreateIndex
CREATE UNIQUE INDEX "AdDemographic_adAccountId_dateFrom_dateTo_campaignId_age_ge_key" ON "AdDemographic"("adAccountId", "dateFrom", "dateTo", "campaignId", "age", "gender");

-- CreateIndex
CREATE INDEX "Report_clientId_createdAt_idx" ON "Report"("clientId", "createdAt");

-- CreateIndex
CREATE INDEX "AiInsight_clientId_module_dateFrom_dateTo_idx" ON "AiInsight"("clientId", "module", "dateFrom", "dateTo");

-- CreateIndex
CREATE INDEX "SyncJob_clientId_startedAt_idx" ON "SyncJob"("clientId", "startedAt");

-- AddForeignKey
ALTER TABLE "Client" ADD CONSTRAINT "Client_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ClientMember" ADD CONSTRAINT "ClientMember_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ClientMember" ADD CONSTRAINT "ClientMember_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Connection" ADD CONSTRAINT "Connection_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SocialAccount" ADD CONSTRAINT "SocialAccount_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SocialAccount" ADD CONSTRAINT "SocialAccount_connectionId_fkey" FOREIGN KEY ("connectionId") REFERENCES "Connection"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SocialSnapshot" ADD CONSTRAINT "SocialSnapshot_socialAccountId_fkey" FOREIGN KEY ("socialAccountId") REFERENCES "SocialAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SocialPost" ADD CONSTRAINT "SocialPost_socialAccountId_fkey" FOREIGN KEY ("socialAccountId") REFERENCES "SocialAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoProperty" ADD CONSTRAINT "SeoProperty_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoProperty" ADD CONSTRAINT "SeoProperty_connectionId_fkey" FOREIGN KEY ("connectionId") REFERENCES "Connection"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoDailyMetric" ADD CONSTRAINT "SeoDailyMetric_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoDimensionMetric" ADD CONSTRAINT "SeoDimensionMetric_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Ga4DailyMetric" ADD CONSTRAINT "Ga4DailyMetric_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoAudit" ADD CONSTRAINT "SeoAudit_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoCrawl" ADD CONSTRAINT "SeoCrawl_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoCrawlPage" ADD CONSTRAINT "SeoCrawlPage_crawlId_fkey" FOREIGN KEY ("crawlId") REFERENCES "SeoCrawl"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SeoIssue" ADD CONSTRAINT "SeoIssue_crawlId_fkey" FOREIGN KEY ("crawlId") REFERENCES "SeoCrawl"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdAccount" ADD CONSTRAINT "AdAccount_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdAccount" ADD CONSTRAINT "AdAccount_connectionId_fkey" FOREIGN KEY ("connectionId") REFERENCES "Connection"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdCampaign" ADD CONSTRAINT "AdCampaign_adAccountId_fkey" FOREIGN KEY ("adAccountId") REFERENCES "AdAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdSet" ADD CONSTRAINT "AdSet_campaignId_fkey" FOREIGN KEY ("campaignId") REFERENCES "AdCampaign"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Ad" ADD CONSTRAINT "Ad_adSetId_fkey" FOREIGN KEY ("adSetId") REFERENCES "AdSet"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdDailyInsight" ADD CONSTRAINT "AdDailyInsight_adAccountId_fkey" FOREIGN KEY ("adAccountId") REFERENCES "AdAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdDailyInsight" ADD CONSTRAINT "AdDailyInsight_campaignId_fkey" FOREIGN KEY ("campaignId") REFERENCES "AdCampaign"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdDailyInsight" ADD CONSTRAINT "AdDailyInsight_adSetId_fkey" FOREIGN KEY ("adSetId") REFERENCES "AdSet"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdDailyInsight" ADD CONSTRAINT "AdDailyInsight_adId_fkey" FOREIGN KEY ("adId") REFERENCES "Ad"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdDemographic" ADD CONSTRAINT "AdDemographic_adAccountId_fkey" FOREIGN KEY ("adAccountId") REFERENCES "AdAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Report" ADD CONSTRAINT "Report_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Report" ADD CONSTRAINT "Report_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AiInsight" ADD CONSTRAINT "AiInsight_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SyncJob" ADD CONSTRAINT "SyncJob_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;
