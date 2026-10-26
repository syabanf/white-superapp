-- +goose Up
-- ERP permission keys per user (pkg/authn/perms.go). Admins hold all keys implicitly.
ALTER TABLE identity.users ADD COLUMN permissions text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE identity.users DROP COLUMN permissions;
