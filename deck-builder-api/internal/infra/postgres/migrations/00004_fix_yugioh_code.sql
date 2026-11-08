-- +goose Up
-- Corrige el typo del código de Yu-Gi-Oh!. Los mazos referencian games.id,
-- así que no hace falta tocar ninguna otra tabla.
UPDATE games SET code = 'yugioh' WHERE code = 'yuigoh';

-- +goose Down
UPDATE games SET code = 'yuigoh' WHERE code = 'yugioh';
