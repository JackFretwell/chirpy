package auth

import (
	"log"
	"time"
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Fatal(err)
	}

	return hash, err
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		log.Fatal(err)
	}

	return match, err
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := CustomClaims{
		jwt.RegisteredClaims{
			Issuer:		"chirpy-access",
			IssuedAt:	jwt.NewNumericDate(time.Now()),
			ExpiresAt:	jwt.NewNumericDate(time.Now().Add(expiresIn)),
			Subject:	userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return ss, err
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error){
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.New(), err
	}

	userID, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.New(), err
	}

	returnedID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.New(), err
	}

	return returnedID, err
}