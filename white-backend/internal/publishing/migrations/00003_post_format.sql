-- +goose Up
-- How a post appears on the platform. FEED keeps the behaviour every
-- existing post had; STORY posts to Instagram/Facebook stories; REEL is a
-- single short video.
ALTER TABLE publishing.content_posts
    ADD COLUMN format text NOT NULL DEFAULT 'FEED'
    CHECK (format IN ('FEED', 'STORY', 'REEL'));

-- +goose Down
ALTER TABLE publishing.content_posts DROP COLUMN format;
