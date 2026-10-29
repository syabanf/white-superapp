-- CreateEnum
CREATE TYPE "AlertSeverity" AS ENUM ('INFO', 'WARNING', 'CRITICAL');

-- CreateEnum
CREATE TYPE "ReportCadence" AS ENUM ('WEEKLY', 'MONTHLY');

-- CreateEnum
CREATE TYPE "CampaignDraftStatus" AS ENUM ('DRAFT', 'READY', 'SUBMITTED');

-- CreateTable
CREATE TABLE "BrandKit" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "primaryColor" TEXT NOT NULL DEFAULT '#2b2265',
    "secondaryColor" TEXT NOT NULL DEFAULT '#fbeaf2',
    "accentColors" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "headingFont" TEXT NOT NULL DEFAULT '',
    "bodyFont" TEXT NOT NULL DEFAULT '',
    "logoUrl" TEXT,
    "toneOfVoice" TEXT NOT NULL DEFAULT '',
    "audience" TEXT NOT NULL DEFAULT '',
    "doList" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "dontList" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "keywords" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "hashtags" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "sampleCaptions" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "visualStyle" TEXT NOT NULL DEFAULT '',
    "notes" TEXT NOT NULL DEFAULT '',
    "updatedById" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "BrandKit_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "DesignDraft" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "brief" TEXT NOT NULL,
    "format" TEXT NOT NULL,
    "layout" JSONB NOT NULL,
    "imagePrompt" TEXT NOT NULL DEFAULT '',
    "imageUrl" TEXT,
    "mediaAssetId" TEXT,
    "model" TEXT NOT NULL,
    "createdById" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "DesignDraft_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AdAlert" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "campaignId" TEXT NOT NULL DEFAULT '',
    "rule" TEXT NOT NULL,
    "severity" "AlertSeverity" NOT NULL,
    "date" DATE NOT NULL,
    "metric" TEXT NOT NULL,
    "current" DOUBLE PRECISION NOT NULL,
    "baseline" DOUBLE PRECISION NOT NULL,
    "message" TEXT NOT NULL,
    "resolvedAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "AdAlert_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "ReportSchedule" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "cadence" "ReportCadence" NOT NULL,
    "modules" "ModuleKey"[] DEFAULT ARRAY['SOCIAL', 'SEO', 'ADS']::"ModuleKey"[],
    "withInsight" BOOLEAN NOT NULL DEFAULT true,
    "enabled" BOOLEAN NOT NULL DEFAULT true,
    "lastRunAt" TIMESTAMP(3),
    "nextRunAt" TIMESTAMP(3) NOT NULL,
    "createdById" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "ReportSchedule_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "CampaignDraft" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "adAccountId" TEXT,
    "name" TEXT NOT NULL,
    "objective" TEXT NOT NULL,
    "dailyBudget" DOUBLE PRECISION NOT NULL,
    "startDate" DATE NOT NULL,
    "endDate" DATE,
    "audience" JSONB NOT NULL,
    "placements" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "creatives" JSONB NOT NULL,
    "aiNotes" TEXT NOT NULL DEFAULT '',
    "status" "CampaignDraftStatus" NOT NULL DEFAULT 'DRAFT',
    "externalId" TEXT,
    "simulated" BOOLEAN NOT NULL DEFAULT false,
    "submittedAt" TIMESTAMP(3),
    "createdById" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "CampaignDraft_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "BrandKit_clientId_key" ON "BrandKit"("clientId");

-- CreateIndex
CREATE INDEX "DesignDraft_clientId_createdAt_idx" ON "DesignDraft"("clientId", "createdAt");

-- CreateIndex
CREATE INDEX "AdAlert_clientId_date_idx" ON "AdAlert"("clientId", "date");

-- CreateIndex
CREATE UNIQUE INDEX "AdAlert_clientId_rule_campaignId_date_key" ON "AdAlert"("clientId", "rule", "campaignId", "date");

-- CreateIndex
CREATE INDEX "ReportSchedule_enabled_nextRunAt_idx" ON "ReportSchedule"("enabled", "nextRunAt");

-- CreateIndex
CREATE INDEX "CampaignDraft_clientId_createdAt_idx" ON "CampaignDraft"("clientId", "createdAt");

-- AddForeignKey
ALTER TABLE "BrandKit" ADD CONSTRAINT "BrandKit_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "DesignDraft" ADD CONSTRAINT "DesignDraft_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AdAlert" ADD CONSTRAINT "AdAlert_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ReportSchedule" ADD CONSTRAINT "ReportSchedule_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "CampaignDraft" ADD CONSTRAINT "CampaignDraft_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;
