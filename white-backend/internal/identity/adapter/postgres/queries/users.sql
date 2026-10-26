-- name: GetUser :one
SELECT * FROM identity.users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM identity.users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM identity.users ORDER BY created_at ASC;

-- name: ListActiveUserOptions :many
SELECT id, name, email FROM identity.users WHERE is_active ORDER BY name ASC;

-- name: ListPermissionHolders :many
SELECT id, name, email, role FROM identity.users
WHERE is_active
  AND (role = 'ADMIN' OR @permission::text = ANY(permissions) OR split_part(@permission::text, '.', 1) || '.*' = ANY(permissions))
ORDER BY name ASC;

-- name: ListUsersByIDs :many
SELECT * FROM identity.users WHERE id = ANY(@ids::text[]);

-- name: InsertUser :one
INSERT INTO identity.users (id, email, name, password_hash, role, is_active, permissions)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateUser :one
UPDATE identity.users
SET name = $2, email = $3, role = $4, is_active = $5, permissions = $6,
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserPermissions :one
UPDATE identity.users SET permissions = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: UpsertLegacyUser :one
INSERT INTO identity.users (id, email, name, password_hash, role, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, name = EXCLUDED.name, password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role, is_active = EXCLUDED.is_active, updated_at = now()
RETURNING *;

-- name: SetUserActive :one
UPDATE identity.users SET is_active = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: DeleteUser :execrows
DELETE FROM identity.users WHERE id = $1;

-- name: ListMembershipsOfUsers :many
SELECT m.user_id, m.client_id, m.role, c.name AS client_name, c.slug AS client_slug
FROM identity.client_members m
JOIN identity.clients c ON c.id = m.client_id
WHERE m.user_id = ANY(@user_ids::text[])
ORDER BY c.name ASC;

-- name: DeleteMembershipsOfUser :exec
DELETE FROM identity.client_members WHERE user_id = $1;

-- name: InsertMembership :exec
INSERT INTO identity.client_members (user_id, client_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, client_id) DO NOTHING;

-- name: UpsertMembership :exec
INSERT INTO identity.client_members (user_id, client_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, client_id) DO UPDATE SET role = EXCLUDED.role;

-- name: UpdateMembershipRole :execrows
UPDATE identity.client_members SET role = $3 WHERE user_id = $1 AND client_id = $2;

-- name: DeleteMembership :execrows
DELETE FROM identity.client_members WHERE user_id = $1 AND client_id = $2;

-- name: GetMembership :one
SELECT * FROM identity.client_members WHERE user_id = $1 AND client_id = $2;

-- name: ListClientMembers :many
SELECT m.user_id, m.role, m.created_at, u.name, u.email, u.is_active
FROM identity.client_members m
JOIN identity.users u ON u.id = m.user_id
WHERE m.client_id = $1
ORDER BY m.created_at ASC;

-- name: ListAudienceUserIDs :many
-- Active admins plus active members of the client (optionally managers only).
SELECT DISTINCT u.id
FROM identity.users u
LEFT JOIN identity.client_members m ON m.user_id = u.id AND m.client_id = @client_id
WHERE u.is_active
  AND (u.role = 'ADMIN' OR (m.user_id IS NOT NULL AND (NOT @managers_only::boolean OR m.role = 'MANAGER')));
