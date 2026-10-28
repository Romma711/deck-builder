package auth

import "testing"

func TestPasswordHasher_RoundTrip(t *testing.T) {
	hasher := NewPasswordHasher(4) // costo bajo para que el test sea rápido

	hash, err := hasher.Hash("supersecreta123")
	if err != nil {
		t.Fatalf("Hash falló: %v", err)
	}
	if hash == "supersecreta123" {
		t.Fatal("la contraseña se devolvió en claro")
	}

	if !hasher.Compare(hash, "supersecreta123") {
		t.Error("Compare debió aceptar la contraseña correcta")
	}
	if hasher.Compare(hash, "otracontraseña") {
		t.Error("Compare debió rechazar la contraseña incorrecta")
	}
}

func TestPasswordHasher_DosHashesDistintosMismaContraseña(t *testing.T) {
	hasher := NewPasswordHasher(4)

	h1, _ := hasher.Hash("mismapassword")
	h2, _ := hasher.Hash("mismapassword")

	// bcrypt usa sal aleatoria: dos hashes del mismo dato no pueden coincidir.
	if h1 == h2 {
		t.Error("los hashes deberían diferir por la sal aleatoria")
	}
}
