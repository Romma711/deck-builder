-- +goose Up
-- ========== CATÁLOGO DE CARTAS NORMALIZADO (Fase 4 multi-TCG) ==========
-- Modelo híbrido: columnas comunes consultables + attributes JSONB con lo
-- específico de cada juego (mana_value/colors de MTG, hp de Pokémon...).

CREATE TABLE cards (
    id          uuid PRIMARY KEY,
    game_id     uuid NOT NULL REFERENCES games(id),
    name        varchar(100) NOT NULL,
    serial_set  varchar(50)  NOT NULL,
    image       text         NOT NULL,
    attributes  jsonb        NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now()
);
-- Identidad de la carta dentro de su juego (para el upsert del catálogo).
CREATE UNIQUE INDEX uq_cards_identity ON cards(game_id, name, serial_set);
CREATE INDEX idx_cards_game_name ON cards(game_id, name);

-- ========== MAZOS → REFERENCIAS AL CATÁLOGO ==========
-- Los decks de esta etapa son datos de desarrollo: se limpian antes de pasar
-- a referencias normalizadas (los datos previos iban embebidos en JSONB).
DELETE FROM decks;

ALTER TABLE decks
    ADD COLUMN game_id         uuid NULL REFERENCES games(id),
    ADD COLUMN leader_card_id  uuid NULL REFERENCES cards(id) ON DELETE SET NULL,
    ADD COLUMN partner_card_id uuid NULL REFERENCES cards(id) ON DELETE SET NULL;

UPDATE decks SET game_id = '10000000-0000-7000-8000-000000000001'; -- fallback: MTG
ALTER TABLE decks ALTER COLUMN game_id SET NOT NULL;
ALTER TABLE decks ADD CONSTRAINT fk_decks_game FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE decks DROP COLUMN leader;
ALTER TABLE decks DROP COLUMN partner;
ALTER TABLE decks DROP COLUMN cards;

DROP INDEX IF EXISTS idx_decks_leader_name;
CREATE INDEX idx_decks_game ON decks(game_id);

-- Cartas del mazo (mazo ↔ catálogo, N:N con cantidad).
CREATE TABLE deck_cards (
    deck_id  uuid NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    card_id  uuid NOT NULL REFERENCES cards(id)  ON DELETE CASCADE,
    quantity int  NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (deck_id, card_id)
);
CREATE INDEX idx_deck_cards_card ON deck_cards(card_id);

-- +goose Down
DROP TABLE IF EXISTS deck_cards;
ALTER TABLE decks DROP COLUMN IF EXISTS game_id;
ALTER TABLE decks DROP COLUMN IF EXISTS leader_card_id;
ALTER TABLE decks DROP COLUMN IF EXISTS partner_card_id;
DROP TABLE IF EXISTS cards;
