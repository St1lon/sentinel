package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/St1lon/sentinel/internal/domain"
)

type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
	issuer string
	now    func() time.Time
}

func NewJWTIssuer(secret string, ttl time.Duration, issuer string) *JWTIssuer {
	return &JWTIssuer{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: issuer,
		now:    time.Now,
	}
}

func (i *JWTIssuer) Issue(userID string) (string, time.Time, error) {
	issuedAt := i.now().UTC()
	expiresAt := issuedAt.Add(i.ttl)

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    i.issuer,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (i *JWTIssuer) ParseUserID(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(
		token,
		&jwt.RegisteredClaims{},
		func(*jwt.Token) (any, error) { return i.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(i.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return "", domain.ErrUnauthenticated
	}

	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", domain.ErrUnauthenticated
	}

	return claims.Subject, nil
}
