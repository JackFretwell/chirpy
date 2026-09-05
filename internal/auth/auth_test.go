package auth

import (
	"testing"
	"time"
	"github.com/google/uuid"
)

//func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error)
//func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) 

func TestValidateJWTPositive(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "devtokensecret"
	expiresIn := 2 * time.Minute
	token, _ := MakeJWT(userID, tokenSecret, expiresIn)
	validatedUserID, _ := ValidateJWT(token, tokenSecret)

	if userID != validatedUserID {
		t.Error("Returned UUID does not match the original")
	}
}

func TestValidateJWTDifferentSecrets(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "devtokensecret"
	differentSecret := "incorrectsecret"
	expiresIn := 2 * time.Minute
	token, _ := MakeJWT(userID, tokenSecret, expiresIn)
	_, err := ValidateJWT(token, differentSecret)

	if err == nil {
		t.Error("Using an incorrect secret should result in an error")
	}
}

func TestValidateJWTExpiredToken(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "devtokensecret"
	differentSecret := "incorrectsecret"
	expiresIn := 1 * time.Second
	token, _ := MakeJWT(userID, tokenSecret, expiresIn)
	time.Sleep(2 * time.Second)
	_, err := ValidateJWT(token, differentSecret)

	if err == nil {
		t.Error("Using an expired token should result in an error")
	}
}

func TestValidateJWTMalformedToken(t *testing.T) {
	_, err := ValidateJWT("not.a.real.token", "devtokensecret")

	if err == nil {
		t.Error("Validating a malformed token should result in an error")
	}
}