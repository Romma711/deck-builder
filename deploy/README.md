# Deploy de Deck Builder (VPS con Docker + Caddy)

Guía para practicar un deploy real: un servidor Linux, Docker Compose y Caddy
con HTTPS automático. Primero se prueba igual en tu máquina; después se repite
en el VPS cambiando solo el `.env`.

```
Internet ──443──▶ caddy ──/api/*──▶ api ──▶ db (Postgres)
                       └──resto───▶ web ──(SSR, red interna)──▶ api
```

| Servicio | Qué es | Expuesto |
|---|---|---|
| `caddy` | Reverse proxy: HTTPS, compresión, headers de seguridad | Puertos 80 y 443 |
| `web` | Front Astro (SSR, Node) | Solo dentro de Docker |
| `api` | API Go; aplica las migraciones al arrancar | Solo `/api/*`, vía Caddy |
| `db` | PostgreSQL 18 | Nada: red interna sin salida a internet |

Decisiones que conviene entender:

- **Mismo origen.** El navegador llama a `https://tu-dominio/api/...`: no hay
  CORS y la imagen del front sirve para cualquier dominio.
- **IP real del visitante.** La API solo confía en `X-Forwarded-For` que venga
  de Caddy o del SSR (IPs fijas). Sin eso, todo el tráfico compartiría la IP
  del proxy y el rate limit sería uno solo para todo el mundo.
- **Rol de base no superusuario.** `postgres-init.sh` crea `deckbuilder` sin
  privilegios de superusuario: es lo que permite que RLS aísle los mazos.
- **`/metrics` y `/readyz` no se publican**: Caddy solo enruta `/api/*`.

---

## 1. Probar en tu máquina

Necesitás los dos repos uno al lado del otro:

```
Trabajo/
├── deck-builder/          ← este repo (con deploy/)
└── deck-builder-front/
```

```bash
cd deck-builder/deploy
cp .env.example .env
```

Generá los tres secretos y pegalos en `.env`:

```bash
openssl rand -base64 48
```

Levantá el stack (la primera vez tarda: compila las dos imágenes):

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

Comprobá que todo esté sano:

```bash
docker compose -f docker-compose.prod.yml ps
```

```bash
curl -k --resolve deckbuilder.localhost:8443:127.0.0.1 https://deckbuilder.localhost:8443/api/v3/games
```

Para navegarlo sin la advertencia de certificado, tu sistema tiene que confiar
en la CA local de Caddy. Es un cambio de seguridad de tu máquina, así que hacelo
solo si querés (y deshacelo al terminar):

```bash
docker compose -f docker-compose.prod.yml cp caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-local-root.crt
```

```bash
sudo trust anchor ./caddy-local-root.crt
```

Para quitarlo después: `sudo trust anchor --remove ./caddy-local-root.crt`.

> En local, el redirect de `http` a `https` apunta al puerto 443 (no al 8443):
> es esperable con puertos no estándar. En el VPS se usan 80 y 443.

Para bajar el stack (los datos quedan en el volumen):

```bash
docker compose -f docker-compose.prod.yml down
```

---

## 2. Preparar el VPS

Cualquier proveedor sirve (Hetzner, DigitalOcean, Linode…). Con 2 GB de RAM
alcanza; compilar las imágenes en el servidor es lo que más memoria usa.

1. **Crear el servidor** con Ubuntu o Debian y tu **clave SSH** (no contraseña).
2. **Usuario sin root** y acceso solo por clave:
   ```bash
   adduser deploy && usermod -aG sudo deploy
   ```
   Copiá tu clave a `~deploy/.ssh/authorized_keys` y en
   `/etc/ssh/sshd_config` poné `PasswordAuthentication no` y
   `PermitRootLogin no`. Reiniciá SSH **sin cerrar la sesión actual** y
   probá entrar desde otra terminal antes de seguir.
3. **Firewall**: solo SSH, HTTP y HTTPS.
   ```bash
   sudo ufw allow OpenSSH && sudo ufw allow 80,443/tcp && sudo ufw allow 443/udp && sudo ufw enable
   ```
   > **Ojo:** Docker publica puertos saltándose `ufw`. Por eso en este stack
   > solo Caddy publica puertos; la base y la API no tienen `ports:`.
4. **Docker**: seguí la instalación oficial de Docker Engine para tu
   distribución y agregá tu usuario al grupo `docker`.
5. **Actualizaciones de seguridad automáticas**:
   ```bash
   sudo apt install unattended-upgrades
   ```

## 3. DNS

En tu proveedor de dominio, creá un registro **A** (y **AAAA** si el VPS tiene
IPv6) apuntando a la IP del servidor. Esperá a que propague:

```bash
dig +short tu-dominio.com
```

Caddy necesita que el dominio ya apunte al servidor para obtener el
certificado de Let's Encrypt.

## 4. Desplegar

En el VPS, cloná los dos repos uno al lado del otro y configurá el `.env`:

```bash
cd deck-builder/deploy && cp .env.example .env
```

Valores de producción:

```env
DOMAIN=tu-dominio.com
SITE_URL=https://tu-dominio.com
HTTP_PORT=80
HTTPS_PORT=443
```

Más los tres secretos (nuevos, nunca los de tu máquina). Protegé el archivo:

```bash
chmod 600 .env
```

Levantá y seguí los logs de Caddy hasta ver el certificado emitido:

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

```bash
docker compose -f docker-compose.prod.yml logs -f caddy
```

## 5. Operación diaria

**Actualizar** a una versión nueva (las migraciones corren solas al
arrancar la API):

```bash
git -C ../ pull && git -C ../../deck-builder-front pull
```

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

**Ver logs** (`api` y `caddy` loguean en JSON):

```bash
docker compose -f docker-compose.prod.yml logs -f --tail 100 api
```

**Limpiar imágenes viejas** después de varios deploys:

```bash
docker image prune -f
```

## 6. Backups

`backup.sh` hace un `pg_dump` en `backups/` y borra los de más de 14 días.
Programalo con `crontab -e` (todos los días a las 4:00):

```
0 4 * * * /home/deploy/deck-builder/deploy/backup.sh >> /home/deploy/backup.log 2>&1
```

Un backup que nunca se restauró no es un backup. Probá restaurar en una base
temporal de vez en cuando:

```bash
docker compose -f docker-compose.prod.yml exec -T db createdb -U postgres restore_check
```

```bash
docker compose -f docker-compose.prod.yml exec -T db pg_restore -U postgres -d restore_check --no-owner < backups/ARCHIVO.dump
```

```bash
docker compose -f docker-compose.prod.yml exec -T db dropdb -U postgres restore_check
```

Además, copiá los backups **fuera del servidor** (otro proveedor, un bucket
S3/B2): si se pierde el VPS, se pierden con él.

## 7. Próximos pasos para practicar

1. **Imágenes en un registry.** Hoy se compila en el servidor. Lo habitual es
   que GitHub Actions construya las imágenes, las suba a GHCR y el servidor
   solo haga `docker compose pull`.
2. **Deploy automático** desde GitHub Actions por SSH al hacer merge a `main`.
3. **Monitoreo**: un chequeo externo de uptime y alertas; la API ya expone
   métricas Prometheus en `/metrics` dentro de la red interna.
4. **Staging**: el mismo stack en otro subdominio para probar antes de
   producción.
