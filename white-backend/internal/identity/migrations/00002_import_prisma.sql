-- +goose Up
-- One-time copy from the Prisma-era tables in `public`, when they exist.
-- Rows keep their ids so references from other services' imports stay valid.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public."User"') IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO identity.users (id, email, name, password_hash, role, is_active, created_at, updated_at)
    SELECT id, email, name, "passwordHash", role::text::identity.user_role, "isActive", "createdAt", "updatedAt"
    FROM public."User"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO identity.clients (id, name, slug, description, logo_url, website_url, industry, currency, timezone,
                                  share_enabled, share_pin_hash, share_modules, created_by_id, created_at, updated_at)
    SELECT id, name, slug, description, "logoUrl", "websiteUrl", industry, currency, timezone,
           "shareEnabled", "sharePinHash", "shareModules"::text[], "createdById", "createdAt", "updatedAt"
    FROM public."Client"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO identity.client_members (user_id, client_id, role, created_at)
    SELECT "userId", "clientId", role::text::identity.client_member_role, "createdAt"
    FROM public."ClientMember"
    ON CONFLICT DO NOTHING;

    INSERT INTO identity.notifications (id, user_id, client_id, type, title, body, href, read_at, created_at)
    SELECT id, "userId", "clientId", type, title, body, href, "readAt", "createdAt"
    FROM public."Notification"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO identity.app_settings (key, value, updated_at)
    SELECT key, value, "updatedAt"
    FROM public."AppSetting"
    ON CONFLICT (key) DO NOTHING;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
