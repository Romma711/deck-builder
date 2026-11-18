-- +goose Up
-- Invitaciones por link a una organización: un solo uso, con vencimiento.
-- Solo se guarda el hash SHA-256 del token (como en refresh_tokens).
CREATE TABLE org_invitations (
    id         uuid        PRIMARY KEY,
    org_id     uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    token_hash char(64)    NOT NULL UNIQUE,
    role       varchar(10) NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    created_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    used_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL
);

CREATE INDEX idx_org_invitations_org ON org_invitations(org_id);

-- +goose Down
DROP TABLE org_invitations;
