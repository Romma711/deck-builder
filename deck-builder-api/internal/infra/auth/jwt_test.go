package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

func TestTokenService_RoundTrip(t *testing.T) {
	svc := NewTokenService("clave-secreta-de-prueba", 15*time.Minute)
	user := &domain.User{ID: "user-1", Username: "romina"}

	token, err := svc.IssueAccessToken(user)
	if err != nil {
		t.Fatalf("IssueAccessToken falló: %v", err)
	}

	claims, err := svc.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken falló: %v", err)
	}
	if claims.UserID != "user-1" || claims.Username != "romina" {
		t.Errorf("claims = %+v, esperaba user-1/romina", claims)
	}
}

func TestTokenService_FirmaInvalida(t *testing.T) {
	svc := NewTokenService("clave-secreta-de-prueba", 15*time.Minute)

	token, _ := svc.IssueAccessToken(&domain.User{ID: "user-1"})

	// Otro servicio con OTRA clave no debe poder parsear nuestro token.
	attacker := NewTokenService("otra-clave", 15*time.Minute)
	if _, err := attacker.ParseAccessToken(token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("token firmado con otra clave debió rechazarse, obtuve %v", err)
	}
}

func TestTokenService_AlgorithmConfusion_Rechazada(t *testing.T) {
	svc := NewTokenService("clave-secreta-de-prueba", 15*time.Minute)

	// Token "firmado" con alg=none: el clásico ataque de confusión de algoritmo.
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub":      "user-1",
		"username": "atacante",
		"iss":      issuer,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	raw, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("no pude fabricar el token none: %v", err)
	}

	if _, err := svc.ParseAccessToken(raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("token 'none' debió rechazarse, obtuve %v", err)
	}
}

func TestTokenService_HS512_Rechazado(t *testing.T) {
	svc := NewTokenService("clave-secreta-de-prueba", 15*time.Minute)

	// Token con otro algoritmo HMAC pero la MISMA clave: si no validamos el
	// método, pasaría. Con WithValidMethods(HS256) debe rechazarse.
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": "user-1",
		"iss": issuer,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	raw, err := token.SignedString([]byte("clave-secreta-de-prueba"))
	if err != nil {
		t.Fatalf("no pude fabricar el token HS512: %v", err)
	}

	if _, err := svc.ParseAccessToken(raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("token HS512 debió rechazarse, obtuve %v", err)
	}
}

func TestTokenService_Expirado(t *testing.T) {
	svc := NewTokenService("clave-secreta-de-prueba", -time.Minute)

	token, _ := svc.IssueAccessToken(&domain.User{ID: "user-1"})
	if _, err := svc.ParseAccessToken(token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("token expirado debió rechazarse, obtuve %v", err)
	}
}

func TestTokenService_ClaimAusente_Sub(t *testing.T) {
	svc := NewTokenService("clave-secreta-de-prueba", time.Minute)

	// Token válido y firmado pero SIN el claim "sub": no debe panickear.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": issuer,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	raw, _ := token.SignedString([]byte("clave-secreta-de-prueba"))

	if _, err := svc.ParseAccessToken(raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("token sin sub debió rechazarse, obtuve %v", err)
	}
}

func TestTokenService_IssuerDistinto(t *testing.T) {
	svc := NewTokenService("clave-secreta-de-prueba", time.Minute)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1",
		"iss": "otro-servicio",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	raw, _ := token.SignedString([]byte("clave-secreta-de-prueba"))

	if _, err := svc.ParseAccessToken(raw); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("token con issuer distinto debió rechazarse, obtuve %v", err)
	}
}
