package postgres

import (
	"fmt"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// embeddedPort evita el 5432, que suele estar ocupado por un Postgres local.
const embeddedPort = 54329

// TestMain prepara la base de los tests de integración:
//   - si TEST_DATABASE_URL ya está definida (CI, contenedor propio), la usa;
//   - si no, levanta un PostgreSQL embebido en embeddedPort.
//
// Si ninguna opción funciona, los tests de integración se omiten (t.Skip) y
// se avisa por stderr para que el "ok" no se confunda con "probado".
func TestMain(m *testing.M) {
	if os.Getenv("TEST_DATABASE_URL") != "" {
		os.Exit(m.Run())
	}

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("test").
		Password("test").
		Database("deckbuilder_test").
		Port(embeddedPort))

	if err := pg.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "AVISO: tests de integración OMITIDOS (sin Postgres embebido: %v)\n", err)
		os.Exit(m.Run())
	}

	os.Setenv("TEST_DATABASE_URL",
		fmt.Sprintf("postgres://test:test@localhost:%d/deckbuilder_test?sslmode=disable", embeddedPort))

	code := m.Run()

	// OJO: esto NO puede ser un defer — TestMain termina en os.Exit y los
	// defers no se ejecutan (dejaba postgres huérfano ocupando el puerto).
	_ = pg.Stop()
	os.Exit(code)
}
