package httpapi

import (
	_ "embed"

	"github.com/gin-gonic/gin"
)

// El spec OpenAPI se embebe en el binario: no hay archivos sueltos que
// desplegar y el spec siempre coincide con la versión del binario.
//
//go:embed openapi.yaml
var openAPISpec []byte

// registerOpenAPIRoute sirve el contrato de la API en /openapi.yaml
// (herramientas como Swagger UI o Redoc lo consumen directamente).
func registerOpenAPIRoute(engine *gin.Engine) {
	engine.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(200, "application/yaml; charset=utf-8", openAPISpec)
	})
}
