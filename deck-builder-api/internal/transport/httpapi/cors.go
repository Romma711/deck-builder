package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// cors implementa CORS configurable por lista de orígenes permitidos.
//
// Comportamiento por defecto (sin orígenes configurados): sin middleware,
// es decir, same-origin — el navegador bloquea peticiones cross-origin.
// NUNCA se responde con "*" + credenciales (aquí no usamos cookies, pero
// mantenemos la lista explícita por seguridad).
func cors(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = true
	}
	if len(allowedOrigins) == 0 {
		// Sin configuración: mismo-origin. Los preflights se contestan 204
		// SIN headers CORS (el navegador bloquea la petición cross-origin);
		// el resto pasa sin tocar nada.
		return func(c *gin.Context) {
			if c.Request.Method == http.MethodOptions &&
				c.GetHeader("Access-Control-Request-Method") != "" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
		}
	}

	const allowHeaders = "Authorization, Content-Type, " + HeaderRequestID

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// El header Vary: Origin debe estar en TODAS las respuestas que
		// puedan variar según el origen (evita caching incorrecto).
		c.Header("Vary", "Origin")

		originAllowed := origin != "" && (allowAll || allowed[origin])
		if originAllowed {
			acao := origin
			if allowAll {
				acao = "*"
			}
			c.Header("Access-Control-Allow-Origin", acao)
			c.Header("Access-Control-Expose-Headers", HeaderRequestID)
		}

		// Preflight: respondemos 204 sin pasar al handler.
		if c.Request.Method == http.MethodOptions &&
			c.GetHeader("Access-Control-Request-Method") != "" {
			if originAllowed {
				c.Header("Access-Control-Allow-Methods",
					"GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", allowHeaders)
				c.Header("Access-Control-Max-Age", "600")
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
