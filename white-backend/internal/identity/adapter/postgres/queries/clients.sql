-- name: GetClient :one
SELECT * FROM identity.clients WHERE id = $1;

-- name: GetClientBySlug :one
SELECT * FROM identity.clients WHERE slug = $1;

-- name: ListAllClients :many
SELECT * FROM identity.clients ORDER BY name ASC;

-- name: ListClientsForUser :many
SELECT c.* FROM identity.clients c
JOIN identity.client_members m ON m.client_id = c.id AND m.user_id = $1
ORDER BY c.name ASC;

-- name: ListClientIDsForUser :many
SELECT client_id FROM identity.client_members WHERE user_id = $1;

-- name: CountMembersByClient :many
SELECT client_id, count(*)::int AS members FROM identity.client_members
WHERE client_id = ANY(@client_ids::text[])
GROUP BY client_id;

-- name: ListMembershipsForUser :many
SELECT client_id, role FROM identity.client_members WHERE user_id = $1;

-- name: InsertClient :one
INSERT INTO identity.clients (id, name, slug, description, industry, website_url, currency, timezone, share_modules, created_by_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateClient :one
UPDATE identity.clients
SET name = $2, slug = $3, description = $4, industry = $5, website_url = $6, currency = $7, timezone = $8, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateShareSettings :one
UPDATE identity.clients
SET share_enabled = $2, share_pin_hash = $3, share_modules = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteClient :execrows
DELETE FROM identity.clients WHERE id = $1;

-- name: CountClients :one
SELECT count(*)::int FROM identity.clients;
