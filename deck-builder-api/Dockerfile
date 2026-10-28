# syntax=docker/dockerfile:1
# Build multi-stage: binario estático (CGO off) + runtime mínimo Alpine con
# usuario no root. El spec OpenAPI y las migraciones viajan embebidos en el
# binario: la imagen no necesita archivos sueltos.

FROM golang:1.27-alpine AS builder
WORKDIR /src

# Capa de dependencias: se cachea mientras go.mod/go.sum no cambien.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22
# ca-certificates: por si la app u operaciones necesitan TLS de salida
# (p. ej. sslmode=require contra un PostgreSQL gestionado).
# wget: lo usa el healthcheck del contenedor.
RUN apk add --no-cache ca-certificates wget \
    && adduser -D -u 10001 app

COPY --from=builder /out/api /usr/local/bin/api

USER app
ENV PORT=8080
EXPOSE 8080

# Sin shell: el healthcheck se define en docker-compose.yml (wget de busybox).
ENTRYPOINT ["/usr/local/bin/api"]
