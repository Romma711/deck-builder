-- +goose Up
-- ========== ORGANIZACIONES Y MEMBRESÍAS (Fase 5 SaaS) ==========
-- El dueño de una org es su membership con rol 'owner' (no hay columna
-- owner_id): así la propiedad se transfiere cambiando roles.

CREATE TABLE organizations (
    id         uuid PRIMARY KEY,
    slug       varchar(50)  NOT NULL UNIQUE,
    name       varchar(100) NOT NULL,
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    org_id     uuid        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       varchar(10) NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id)
);
CREATE INDEX idx_memberships_user ON memberships(user_id);

-- ========== MAZOS: TENANCY + VISIBILIDAD ==========

ALTER TABLE decks
    ADD COLUMN org_id     uuid NULL REFERENCES organizations(id) ON DELETE SET NULL,
    ADD COLUMN is_private boolean NOT NULL DEFAULT false; -- default = público (comportamiento previo)
CREATE INDEX idx_decks_org ON decks(org_id);

-- ========== ROW-LEVEL SECURITY (RLS) ==========
-- La app setea 'app.user_id' con SET LOCAL dentro de cada transacción de
-- decks (ver DeckRepository.withRLS). Vacío/ausente = anónimo.
--
-- IMPORTANTE (ops): RLS solo aplica a roles NO superusuarios. Conexión en
-- producción = rol no superusuario (FORCE cubre al propietario de las
-- tablas). Migraciones futuras que actualicen decks como propietario deben
-- hacerlo con 'SET row_security = off' o verán 0 filas.

ALTER TABLE decks ENABLE ROW LEVEL SECURITY;
ALTER TABLE decks FORCE ROW LEVEL SECURITY;

-- Lectura: público, o propio, o de una org a la que pertenezco.
CREATE POLICY decks_select ON decks FOR SELECT USING (
    is_private = false
    OR owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid
    OR (
        org_id IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM memberships m
            WHERE m.org_id = decks.org_id
              AND m.user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
        )
    )
);

-- Alta: solo como dueño, y si es de org debo ser miembro. (Refuerzo de la
-- validación de aplicación: si la app olvida un chequeo, la BD no cede.)
CREATE POLICY decks_insert ON decks FOR INSERT WITH CHECK (
    owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid
    AND (
        org_id IS NULL
        OR EXISTS (
            SELECT 1 FROM memberships m
            WHERE m.org_id = decks.org_id
              AND m.user_id = NULLIF(current_setting('app.user_id', true), '')::uuid
        )
    )
);

-- Actualización: solo el dueño de la fila (USING filtra filas ajenas y
-- WITH CHECK impide reapropiarse el mazo cambiando owner_id).
CREATE POLICY decks_update ON decks FOR UPDATE
    USING      (owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

CREATE POLICY decks_delete ON decks FOR DELETE USING (
    owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid
);

-- Las cartas del mazo heredan la visibilidad del mazo (la subquery a decks
-- aplica las políticas de arriba) y solo su dueño puede escribirlas.
ALTER TABLE deck_cards ENABLE ROW LEVEL SECURITY;
ALTER TABLE deck_cards FORCE ROW LEVEL SECURITY;

CREATE POLICY deck_cards_select ON deck_cards FOR SELECT USING (
    EXISTS (SELECT 1 FROM decks d WHERE d.id = deck_id)
);

CREATE POLICY deck_cards_write ON deck_cards FOR INSERT WITH CHECK (
    EXISTS (
        SELECT 1 FROM decks d
        WHERE d.id = deck_id
          AND d.owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid
    )
);

CREATE POLICY deck_cards_update ON deck_cards FOR UPDATE
    USING      (EXISTS (
        SELECT 1 FROM decks d
        WHERE d.id = deck_id
          AND d.owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid
    ))
    WITH CHECK (EXISTS (
        SELECT 1 FROM decks d
        WHERE d.id = deck_id
          AND d.owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid
    ));

CREATE POLICY deck_cards_delete ON deck_cards FOR DELETE USING (
    EXISTS (
        SELECT 1 FROM decks d
        WHERE d.id = deck_id
          AND d.owner_id = NULLIF(current_setting('app.user_id', true), '')::uuid
    )
);

-- +goose Down
ALTER TABLE deck_cards DISABLE ROW LEVEL SECURITY;
ALTER TABLE decks DISABLE ROW LEVEL SECURITY;
ALTER TABLE decks DROP COLUMN IF EXISTS org_id;
ALTER TABLE decks DROP COLUMN IF EXISTS is_private;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS organizations;
