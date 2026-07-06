package token

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func mustGenerateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	return key
}

func TestGenerateAndParseUserID(t *testing.T) {
	key := mustGenerateKey(t)

	tok, err := Generate(42, key, time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	userID, err := ParseUserID(tok, &key.PublicKey)
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
}

func TestParseUserIDExpired(t *testing.T) {
	key := mustGenerateKey(t)

	tok, err := Generate(1, key, -time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = ParseUserID(tok, &key.PublicKey)
	if err == nil {
		t.Fatal("ParseUserID() error = nil, want expiry error")
	}
}

func TestParseUserIDWrongKey(t *testing.T) {
	key := mustGenerateKey(t)
	otherKey := mustGenerateKey(t)

	tok, err := Generate(1, key, time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, err = ParseUserID(tok, &otherKey.PublicKey)
	if err == nil {
		t.Fatal("ParseUserID() error = nil, want signature error")
	}
}

func TestParseUserIDRejectsNonRSASigningMethod(t *testing.T) {
	key := mustGenerateKey(t)

	claims := Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	// Sign with an HMAC method to simulate an alg-substitution attack.
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, err = ParseUserID(tok, &key.PublicKey)
	if err == nil {
		t.Fatal("ParseUserID() error = nil, want unexpected signing method error")
	}
}

func TestParseUserIDMalformedToken(t *testing.T) {
	key := mustGenerateKey(t)

	_, err := ParseUserID("not-a-valid-token", &key.PublicKey)
	if err == nil {
		t.Fatal("ParseUserID() error = nil, want parse error")
	}
}

func TestGenerateSetsIssuer(t *testing.T) {
	key := mustGenerateKey(t)

	tok, err := Generate(1, key, time.Hour)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	parsed, _, err := jwt.NewParser().ParseUnverified(tok, &Claims{})
	if err != nil {
		t.Fatalf("ParseUnverified() error = %v", err)
	}
	claims := parsed.Claims.(*Claims)
	if claims.Issuer != Issuer {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, Issuer)
	}
}
