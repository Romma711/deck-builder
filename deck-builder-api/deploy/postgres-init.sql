-- Init de docker-compose.yml (se ejecuta UNA vez, en el primer arranque).
--
-- Crea un rol NO superusuario dueño de la BD: es la condición para que
-- RLS (migración 00003) aísle de verdad los mazos privados y de org.
-- Si cambiás la contraseña, actualizá también DATABASE_URL en docker-compose.yml.

CREATE ROLE deckbuilder LOGIN PASSWORD 'deckbuilder';
CREATE DATABASE deckbuilder OWNER deckbuilder;
