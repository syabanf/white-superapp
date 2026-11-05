-- +goose Up
-- Every change to Finance: who, what they sent, what came back, and refusals too.
CREATE TABLE finance.audit_trail (
  id         text PRIMARY KEY,
  at         timestamptz NOT NULL DEFAULT now(),
  actor_id   text NOT NULL DEFAULT '',
  actor_name text NOT NULL DEFAULT '',
  action     text NOT NULL,
  doc_id     text NOT NULL DEFAULT '',
  doc_no     text NOT NULL DEFAULT '',
  result     text NOT NULL CHECK (result IN ('OK', 'REFUSED')),
  message    text NOT NULL DEFAULT '',
  request    text NOT NULL DEFAULT '',
  response   text NOT NULL DEFAULT ''
);
CREATE INDEX audit_trail_at ON finance.audit_trail (at DESC);
CREATE INDEX audit_trail_doc ON finance.audit_trail (doc_id) WHERE doc_id <> '';

-- An audit trail nobody can rewrite, the application included.
-- +goose StatementBegin
CREATE FUNCTION finance.audit_trail_append_only() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'audit trail tidak bisa diubah atau dihapus';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER audit_trail_append_only BEFORE UPDATE OR DELETE ON finance.audit_trail
  FOR EACH ROW EXECUTE FUNCTION finance.audit_trail_append_only();

-- +goose Down
DROP TABLE finance.audit_trail;
DROP FUNCTION finance.audit_trail_append_only();
