package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

// HeaderRequestID es el header estándar de correlación de peticiones.
const HeaderRequestID = "X-Request-ID"

// keyRequestID guarda el request ID en el contexto de gin.
const keyRequestID = "request_id"

// maxRequestIDLen limita el largo aceptado de un request ID entrante para
// evitar inyección de logs / abuso de memoria.
const maxRequestIDLen = 64

// requestID garantiza que toda respuesta lleve un identificador correlacionable.
//
// Reglas:
//   - Si el cliente envía X-Request-ID válido (alfanumérico + "-" + "_"),
//     se respeta (correlación con el cliente).
//   - En cualquier otro caso se genera uno nuevo (hex aleatorio de 16 bytes).
//
// El valor se expone en la respuesta y en los logs estructurados.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(HeaderRequestID))
		if !isValidRequestID(id) {
			id = newRequestID()
		}
		c.Set(keyRequestID, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// RequestID devuelve el ID de la petición actual (vacío si no pasó por el
// middleware requestID).
func RequestID(c *gin.Context) string {
	return c.GetString(keyRequestID)
}

func isValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown" // crypto/rand fallando: situación imposible en la práctica
	}
	return hex.EncodeToString(b[:])
}
