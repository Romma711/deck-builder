package domain

import "time"

// User es la entidad de usuario del dominio. No conoce BSON, GORM ni JSON:
// solo datos de negocio. El ID es opaco (string) para no atar el dominio al
// esquema de persistencia (ObjectID hoy, UUID/bigserial en la Fase 3).
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
