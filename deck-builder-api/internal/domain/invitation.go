package domain

import "time"

// InvitationTTL es la vigencia de un link de invitación.
const InvitationTTL = 7 * 24 * time.Hour

// Invitation es un link de un solo uso para sumarse a una organización.
// El token en claro solo existe al crearla: se persiste su hash.
type Invitation struct {
	ID        string
	OrgID     string
	Role      Role
	CreatedBy string
	ExpiresAt time.Time
	CreatedAt time.Time

	// Datos de lectura (JOIN), para mostrar la invitación.
	OrgName     string
	OrgSlug     string
	InviterName string
}
