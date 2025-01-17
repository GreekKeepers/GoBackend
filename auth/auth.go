package auth

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/blake2b"
	"greekkeepers.io/backend/responses"
)

func CreateCredentials(
	sub string,
	iss string,
	bearerExp uint64,
	refreshExp uint64,
	secret []byte,
) (*responses.Credentials, error) {
	now := time.Now()
	bearer_token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{"iss": iss, "sub": sub, "exp": now.Add(time.Second * time.Duration(bearerExp)).Unix(), "iat": now.Unix(), "aud": "auth"})
	refresh_token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{"iss": iss, "sub": sub, "exp": now.Add(time.Second * time.Duration(refreshExp)).Unix(), "iat": now.Unix(), "aud": "refresh"})

	bearerString, err := bearer_token.SignedString(secret)
	if err != nil {
		return nil, err
	}
	refreshString, err := refresh_token.SignedString(secret)
	if err != nil {
		return nil, err
	}

	return &responses.Credentials{
		AccessToken:  bearerString,
		RefreshToken: refreshString,
		TokenType:    "Bearer",
		ExpiresIn:    bearerExp,
	}, nil
}

func VerifyToken(tokenString string, secret []byte) (jwt.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("Malformed token")
	}

	exp_time, err := token.Claims.GetExpirationTime()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if now.Unix() > exp_time.Time.Unix() {
		return nil, errors.New("Token expired")
	}

	return token.Claims, nil
}

func HashPassword(password string, salt string) string {
	hash := blake2b.Sum256([]byte(password + salt))

	return hex.EncodeToString(hash[:])
}
