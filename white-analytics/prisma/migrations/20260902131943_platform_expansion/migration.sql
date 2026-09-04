-- CreateEnum
CREATE TYPE "PostStatus" AS ENUM ('DRAFT', 'IN_REVIEW', 'APPROVED', 'SCHEDULED', 'PUBLISHING', 'PUBLISHED', 'FAILED', 'REJECTED');

-- CreateEnum
CREATE TYPE "MediaKind" AS ENUM ('IMAGE', 'VIDEO');

-- CreateEnum
CREATE TYPE "DeviceType" AS ENUM ('DESKTOP', 'MOBILE');

-- CreateEnum
CREATE TYPE "KeywordIntent" AS ENUM ('INFORMATIONAL', 'NAVIGATIONAL', 'COMMERCIAL', 'TRANSACTIONAL');

-- CreateEnum
CREATE TYPE "AuditCadence" AS ENUM ('NONE', 'WEEKLY', 'DAILY');

-- AlterTable
ALTER TABLE "SeoProperty" ADD COLUMN     "auditCadence" "AuditCadence" NOT NULL DEFAULT 'NONE',
ADD COLUMN     "languageCode" TEXT NOT NULL DEFAULT 'id',
ADD COLUMN     "locationCode" INTEGER NOT NULL DEFAULT 2360;

-- CreateTable
CREATE TABLE "MediaAsset" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "kind" "MediaKind" NOT NULL,
    "url" TEXT NOT NULL,
    "thumbnailUrl" TEXT,
    "filename" TEXT NOT NULL,
    "mimeType" TEXT NOT NULL,
    "sizeBytes" INTEGER NOT NULL DEFAULT 0,
    "width" INTEGER,
    "height" INTEGER,
    "durationSec" DOUBLE PRECISION,
    "altText" TEXT,
    "tags" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "uploadedById" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "MediaAsset_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "ContentPost" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "title" TEXT NOT NULL DEFAULT '',
    "body" TEXT NOT NULL DEFAULT '',
    "status" "PostStatus" NOT NULL DEFAULT 'DRAFT',
    "scheduledAt" TIMESTAMP(3),
    "publishedAt" TIMESTAMP(3),
    "linkUrl" TEXT,
    "firstComment" TEXT,
    "labels" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "utm" JSONB,
    "reviewNote" TEXT,
    "createdById" TEXT,
    "approvedById" TEXT,
    "approvedAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "ContentPost_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "PostTarget" (
    "id" TEXT NOT NULL,
    "postId" TEXT NOT NULL,
    "socialAccountId" TEXT NOT NULL,
    "bodyOverride" TEXT,
    "status" "PostStatus" NOT NULL DEFAULT 'DRAFT',
    "externalId" TEXT,
    "permalink" TEXT,
    "error" TEXT,
    "attempts" INTEGER NOT NULL DEFAULT 0,
    "publishedAt" TIMESTAMP(3),

    CONSTRAINT "PostTarget_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "PostMedia" (
    "postId" TEXT NOT NULL,
    "assetId" TEXT NOT NULL,
    "order" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "PostMedia_pkey" PRIMARY KEY ("postId","assetId")
);

-- CreateTable
CREATE TABLE "PostComment" (
    "id" TEXT NOT NULL,
    "postId" TEXT NOT NULL,
    "userId" TEXT,
    "body" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "PostComment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "PostActivity" (
    "id" TEXT NOT NULL,
    "postId" TEXT NOT NULL,
    "userId" TEXT,
    "type" TEXT NOT NULL,
    "meta" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "PostActivity_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TrackedKeyword" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "keyword" TEXT NOT NULL,
    "locationCode" INTEGER NOT NULL DEFAULT 2360,
    "languageCode" TEXT NOT NULL DEFAULT 'id',
    "device" "DeviceType" NOT NULL DEFAULT 'MOBILE',
    "tags" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "targetUrl" TEXT,
    "volume" INTEGER,
    "difficulty" INTEGER,
    "cpc" DOUBLE PRECISION,
    "intent" "KeywordIntent",
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "TrackedKeyword_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "RankSnapshot" (
    "id" TEXT NOT NULL,
    "keywordId" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "position" INTEGER,
    "url" TEXT,
    "serpFeatures" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "topResults" JSONB,

    CONSTRAINT "RankSnapshot_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "KeywordResearch" (
    "id" TEXT NOT NULL,
    "clientId" TEXT NOT NULL,
    "seed" TEXT NOT NULL,
    "locationCode" INTEGER NOT NULL DEFAULT 2360,
    "languageCode" TEXT NOT NULL DEFAULT 'id',
    "provider" TEXT NOT NULL,
    "results" JSONB NOT NULL,
    "fetchedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "KeywordResearch_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "BacklinkSnapshot" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "backlinks" INTEGER NOT NULL,
    "referringDomains" INTEGER NOT NULL,
    "dofollow" INTEGER NOT NULL DEFAULT 0,
    "nofollow" INTEGER NOT NULL DEFAULT 0,
    "newBacklinks" INTEGER NOT NULL DEFAULT 0,
    "lostBacklinks" INTEGER NOT NULL DEFAULT 0,
    "domainRank" INTEGER,
    "toxicShare" DOUBLE PRECISION,

    CONSTRAINT "BacklinkSnapshot_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Backlink" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "sourceUrl" TEXT NOT NULL,
    "sourceDomain" TEXT NOT NULL,
    "targetUrl" TEXT NOT NULL,
    "anchor" TEXT NOT NULL DEFAULT '',
    "dofollow" BOOLEAN NOT NULL DEFAULT true,
    "domainRank" INTEGER,
    "spamScore" INTEGER,
    "firstSeen" DATE NOT NULL,
    "lastSeen" DATE NOT NULL,
    "isLost" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "Backlink_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "CompetitorDomain" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "domain" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "CompetitorDomain_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "DomainSnapshot" (
    "id" TEXT NOT NULL,
    "propertyId" TEXT NOT NULL,
    "domain" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "organicKeywords" INTEGER NOT NULL,
    "organicTraffic" INTEGER NOT NULL,
    "organicCost" DOUBLE PRECISION,
    "top3" INTEGER NOT NULL DEFAULT 0,
    "top10" INTEGER NOT NULL DEFAULT 0,
    "top100" INTEGER NOT NULL DEFAULT 0,
    "backlinks" INTEGER,
    "referringDomains" INTEGER,
    "domainRank" INTEGER,

    CONSTRAINT "DomainSnapshot_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Notification" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "clientId" TEXT,
    "type" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "body" TEXT NOT NULL DEFAULT '',
    "href" TEXT,
    "readAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Notification_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "MediaAsset_clientId_createdAt_idx" ON "MediaAsset"("clientId", "createdAt");

-- CreateIndex
CREATE INDEX "ContentPost_clientId_status_scheduledAt_idx" ON "ContentPost"("clientId", "status", "scheduledAt");

-- CreateIndex
CREATE INDEX "ContentPost_clientId_scheduledAt_idx" ON "ContentPost"("clientId", "scheduledAt");

-- CreateIndex
CREATE INDEX "PostTarget_status_idx" ON "PostTarget"("status");

-- CreateIndex
CREATE UNIQUE INDEX "PostTarget_postId_socialAccountId_key" ON "PostTarget"("postId", "socialAccountId");

-- CreateIndex
CREATE INDEX "PostMedia_postId_order_idx" ON "PostMedia"("postId", "order");

-- CreateIndex
CREATE INDEX "PostComment_postId_createdAt_idx" ON "PostComment"("postId", "createdAt");

-- CreateIndex
CREATE INDEX "PostActivity_postId_createdAt_idx" ON "PostActivity"("postId", "createdAt");

-- CreateIndex
CREATE INDEX "TrackedKeyword_propertyId_idx" ON "TrackedKeyword"("propertyId");

-- CreateIndex
CREATE UNIQUE INDEX "TrackedKeyword_propertyId_keyword_locationCode_device_key" ON "TrackedKeyword"("propertyId", "keyword", "locationCode", "device");

-- CreateIndex
CREATE INDEX "RankSnapshot_keywordId_date_idx" ON "RankSnapshot"("keywordId", "date");

-- CreateIndex
CREATE UNIQUE INDEX "RankSnapshot_keywordId_date_key" ON "RankSnapshot"("keywordId", "date");

-- CreateIndex
CREATE INDEX "KeywordResearch_clientId_fetchedAt_idx" ON "KeywordResearch"("clientId", "fetchedAt");

-- CreateIndex
CREATE INDEX "KeywordResearch_clientId_seed_locationCode_languageCode_idx" ON "KeywordResearch"("clientId", "seed", "locationCode", "languageCode");

-- CreateIndex
CREATE INDEX "BacklinkSnapshot_propertyId_date_idx" ON "BacklinkSnapshot"("propertyId", "date");

-- CreateIndex
CREATE UNIQUE INDEX "BacklinkSnapshot_propertyId_date_key" ON "BacklinkSnapshot"("propertyId", "date");

-- CreateIndex
CREATE INDEX "Backlink_propertyId_isLost_lastSeen_idx" ON "Backlink"("propertyId", "isLost", "lastSeen");

-- CreateIndex
CREATE INDEX "Backlink_propertyId_sourceDomain_idx" ON "Backlink"("propertyId", "sourceDomain");

-- CreateIndex
CREATE UNIQUE INDEX "Backlink_propertyId_sourceUrl_targetUrl_key" ON "Backlink"("propertyId", "sourceUrl", "targetUrl");

-- CreateIndex
CREATE UNIQUE INDEX "CompetitorDomain_propertyId_domain_key" ON "CompetitorDomain"("propertyId", "domain");

-- CreateIndex
CREATE INDEX "DomainSnapshot_propertyId_domain_date_idx" ON "DomainSnapshot"("propertyId", "domain", "date");

-- CreateIndex
CREATE UNIQUE INDEX "DomainSnapshot_propertyId_domain_date_key" ON "DomainSnapshot"("propertyId", "domain", "date");

-- CreateIndex
CREATE INDEX "Notification_userId_readAt_createdAt_idx" ON "Notification"("userId", "readAt", "createdAt");

-- AddForeignKey
ALTER TABLE "MediaAsset" ADD CONSTRAINT "MediaAsset_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "MediaAsset" ADD CONSTRAINT "MediaAsset_uploadedById_fkey" FOREIGN KEY ("uploadedById") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ContentPost" ADD CONSTRAINT "ContentPost_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ContentPost" ADD CONSTRAINT "ContentPost_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ContentPost" ADD CONSTRAINT "ContentPost_approvedById_fkey" FOREIGN KEY ("approvedById") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostTarget" ADD CONSTRAINT "PostTarget_postId_fkey" FOREIGN KEY ("postId") REFERENCES "ContentPost"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostTarget" ADD CONSTRAINT "PostTarget_socialAccountId_fkey" FOREIGN KEY ("socialAccountId") REFERENCES "SocialAccount"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostMedia" ADD CONSTRAINT "PostMedia_postId_fkey" FOREIGN KEY ("postId") REFERENCES "ContentPost"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostMedia" ADD CONSTRAINT "PostMedia_assetId_fkey" FOREIGN KEY ("assetId") REFERENCES "MediaAsset"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostComment" ADD CONSTRAINT "PostComment_postId_fkey" FOREIGN KEY ("postId") REFERENCES "ContentPost"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostComment" ADD CONSTRAINT "PostComment_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostActivity" ADD CONSTRAINT "PostActivity_postId_fkey" FOREIGN KEY ("postId") REFERENCES "ContentPost"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PostActivity" ADD CONSTRAINT "PostActivity_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TrackedKeyword" ADD CONSTRAINT "TrackedKeyword_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "RankSnapshot" ADD CONSTRAINT "RankSnapshot_keywordId_fkey" FOREIGN KEY ("keywordId") REFERENCES "TrackedKeyword"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "KeywordResearch" ADD CONSTRAINT "KeywordResearch_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "BacklinkSnapshot" ADD CONSTRAINT "BacklinkSnapshot_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Backlink" ADD CONSTRAINT "Backlink_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "CompetitorDomain" ADD CONSTRAINT "CompetitorDomain_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "DomainSnapshot" ADD CONSTRAINT "DomainSnapshot_propertyId_fkey" FOREIGN KEY ("propertyId") REFERENCES "SeoProperty"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Notification" ADD CONSTRAINT "Notification_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Notification" ADD CONSTRAINT "Notification_clientId_fkey" FOREIGN KEY ("clientId") REFERENCES "Client"("id") ON DELETE CASCADE ON UPDATE CASCADE;
