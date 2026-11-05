-- +goose Up
-- ========== USUARIOS Y SaaS ==========
CREATE TABLE users (
    id            uuid PRIMARY KEY,
    username      varchar(30)  NOT NULL UNIQUE,
    email         varchar(255) NOT NULL UNIQUE,
    password_hash varchar(255) NOT NULL,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash char(64) NOT NULL UNIQUE, -- SHA-256 hex
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user    ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);

-- ========== CATÁLOGO MULTI-TCG (seeds; GameSpec en Fase 4) ==========
CREATE TABLE games (
    id     uuid PRIMARY KEY,
    code   varchar(20)  NOT NULL UNIQUE,
    name   varchar(100) NOT NULL,
    active boolean NOT NULL DEFAULT true
);

CREATE TABLE formats (
    id              uuid PRIMARY KEY,
    game_id         uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    code            varchar(30) NOT NULL,
    name            varchar(100) NOT NULL,
    deck_size       int NULL,
    min_size        int NULL,
    max_size        int NULL,
    requires_leader boolean NOT NULL DEFAULT false,
    UNIQUE (game_id, code)
);

-- Juegos soportados (IDs deterministas para facilitar referencias)
INSERT INTO games (id, code, name) VALUES
    ('10000000-0000-7000-8000-000000000001', 'mtg',      'Magic: The Gathering'),
    ('10000000-0000-7000-8000-000000000002', 'pokemon',  'Pokémon TCG'),
    ('10000000-0000-7000-8000-000000000003', 'yuigoh',   'Yu-Gi-Oh!'),
    ('10000000-0000-7000-8000-000000000004', 'onepiece', 'One Piece TCG');

-- Formatos (semilla inicial; las reglas finas viven en GameSpec/Fase 4)
INSERT INTO formats (id, game_id, code, name, deck_size, min_size, max_size, requires_leader) VALUES
    -- MTG
    ('20000000-0000-7000-8000-000000000001', '10000000-0000-7000-8000-000000000001', 'commander', 'Commander', 100, 100, 100, true),
    ('20000000-0000-7000-8000-000000000002', '10000000-0000-7000-8000-000000000001', 'standard',  'Standard',  60,  60,  60, false),
    ('20000000-0000-7000-8000-000000000003', '10000000-0000-7000-8000-000000000001', 'modern',    'Modern',    60,  60,  60, false),
    ('20000000-0000-7000-8000-000000000004', '10000000-0000-7000-8000-000000000001', 'pioneer',   'Pioneer',   60,  60,  60, false),
    ('20000000-0000-7000-8000-000000000005', '10000000-0000-7000-8000-000000000001', 'legacy',    'Legacy',    60,  60,  60, false),
    ('20000000-0000-7000-8000-000000000006', '10000000-0000-7000-8000-000000000001', 'vintage',   'Vintage',   60,  60,  60, false),
    ('20000000-0000-7000-8000-000000000007', '10000000-0000-7000-8000-000000000001', 'brawl',     'Brawl',      0,  60, 100, true),
    ('20000000-0000-7000-8000-000000000008', '10000000-0000-7000-8000-000000000001', 'historic',  'Historic',  60,  60,  60, false),
    ('20000000-0000-7000-8000-000000000009', '10000000-0000-7000-8000-000000000001', 'pauper',    'Pauper',    60,  60,  60, false),
    -- Pokémon
    ('20000000-0000-7000-8000-000000000010', '10000000-0000-7000-8000-000000000002', 'standard',  'Standard',  60,  60,  60, false),
    ('20000000-0000-7000-8000-000000000011', '10000000-0000-7000-8000-000000000002', 'expanded',  'Expanded',  60,  60,  60, false),
    -- Yu-Gi-Oh!
    ('20000000-0000-7000-8000-000000000012', '10000000-0000-7000-8000-000000000003', 'standard',  'Standard',  40,  40,  60, false),
    -- One Piece
    ('20000000-0000-7000-8000-000000000013', '10000000-0000-7000-8000-000000000004', 'standard',  'STANDARD',  50,  50,  50, true),
    ('20000000-0000-7000-8000-000000000014', '10000000-0000-7000-8000-000000000004', 'rookie',    'ROOKIE',    30,   1,  50, true);

-- ========== MAZOS ==========
-- TRANSITORIO respecto al modelo normalizado de la Fase 4: las cartas van
-- embebidas en JSONB (equivalente al documento de Mongo). La Fase 4 creará
-- cards/deck_cards normalizadas con la migración 00002.
CREATE TABLE decks (
    id          uuid PRIMARY KEY,
    owner_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        varchar(50)  NOT NULL,
    description varchar(300) NOT NULL DEFAULT '',
    deck_size   int NOT NULL,
    format      varchar(30)  NOT NULL,
    leader      jsonb NULL,
    partner     jsonb NULL,
    cards       jsonb NOT NULL DEFAULT '[]',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_decks_owner   ON decks(owner_id);
CREATE INDEX idx_decks_format  ON decks(format);
CREATE INDEX idx_decks_leader_name ON decks ((leader->>'name'));

-- +goose Down
DROP TABLE IF EXISTS decks;
DROP TABLE IF EXISTS formats;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
