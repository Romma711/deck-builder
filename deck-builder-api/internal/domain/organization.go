package domain

import "time"

// Role es el rol de un usuario dentro de una organización (jerarquía
// owner > admin > member).
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// ValidRole comprueba que el rol esté en el conjunto soportado.
func ValidRole(r Role) bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}

// AtLeast evalúa la jerarquía: owner ≥ admin ≥ member. Falla CERRADA: un
// rol desconocido (ya sea el que actúa o el mínimo pedido) nunca satisface
// la condición.
func (r Role) AtLeast(min Role) bool {
	rank := map[Role]int{RoleMember: 1, RoleAdmin: 2, RoleOwner: 3}
	rRank, okR := rank[r]
	minRank, okMin := rank[min]
	return okR && okMin && rRank >= minRank
}

// Organization es una organización (tienda o grupo) del SaaS.
//
// El "dueño" NO es una columna: es la membership con rol owner (permite
// transferir la propiedad cambiando roles sin tocar la org).
type Organization struct {
	ID   string
	Slug string // identificador URL amigable, único
	Name string
	// Role es el rol del usuario consultante; solo se llena en listados por
	// usuario (ListByUser). No es estado de la organización.
	Role      Role
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Membership liga un usuario a una organización con un rol.
type Membership struct {
	OrgID     string
	UserID    string
	Role      Role
	CreatedAt time.Time
}
