package postgres

import (
	"os"
	"os/exec"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// TestMain levanta un PostgreSQL embebido para los tests de integración.
func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("test").
		Password("test").
		Database("deckbuilder_test"))

	if err := pg.Start(); err != nil {
		// Sin red/binarios disponibles: los tests de integración se omiten.
		os.Exit(m.Run())
	}

	os.Setenv("TEST_DATABASE_URL",
		"postgres://test:test@localhost:5432/deckbuilder_test?sslmode=disable")

	code := m.Run()

	// OJO: esto NO puede ser un defer — TestMain termina en os.Exit y los
	// defers no se ejecutan (dejaba postgres huérfano ocupando el 5432).
	_ = pg.Stop()

	if code != 0 {
		exec.Command("sh", "-c", "echo 'FAILS en integración'").Run()
	}
	os.Exit(code)
}
