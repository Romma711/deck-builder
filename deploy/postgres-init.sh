#!/bin/sh
# Primer arranque de Postgres (volumen vacío): crea el rol de la app y su base.
# El rol NO es superusuario: es la condición para que RLS aísle los mazos
# privados y de organizaciones.
set -eu

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -v app_password="$APP_DB_PASSWORD" <<'SQL'
CREATE ROLE deckbuilder LOGIN PASSWORD :'app_password';
CREATE DATABASE deckbuilder OWNER deckbuilder;
SQL
