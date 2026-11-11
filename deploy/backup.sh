#!/bin/sh
# Backup de la base con pg_dump (formato custom, comprimido).
# Uso: ./backup.sh            (desde deploy/; pensado para cron)
# Restaurar: ver README.md, sección "Backups".
set -eu

cd "$(dirname "$0")"
mkdir -p backups
file="backups/deckbuilder-$(date +%Y%m%d-%H%M%S).dump"

docker compose -f docker-compose.prod.yml exec -T db \
  pg_dump -U postgres -Fc deckbuilder > "$file"

# Retención: se borran los backups de más de 14 días.
find backups -name 'deckbuilder-*.dump' -mtime +14 -delete
echo "backup: $file"
