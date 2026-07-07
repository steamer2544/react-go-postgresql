package service

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT payload for an access token.
type Claims struct {
	UserID    uint      `json:"sub"`
	Role      string    `json:"role"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// TokenService issues and verifies JWT access tokens.
type TokenService interface {
	Issue(userID uint, role string) (string, error)
	Verify(tokenString string) (Claims, error)
}

type tokenService struct {
	secret string
	ttl    time.Duration
}

// NewTokenService creates a TokenService with the given HS256 secret and token TTL.
func NewTokenService(secret string, ttl time.Duration) TokenService {
	return &tokenService{secret: secret, ttl: ttl}
}

// Issue creates a signed JWT containing sub, role, exp, iat.
func (s *tokenService) Issue(userID uint, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  strconv.Itoa(int(userID)),
		"role": role,
		"exp":  now.Add(s.ttl).Unix(),
		"iat":  now.Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.secret))
}

// Verify parses and validates a JWT string, returning the typed Claims.
func (s *tokenService) Verify(tokenString string) (Claims, error) {
	tok, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return Claims{}, err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return Claims{}, jwt.ErrSignatureInvalid
	}
	subStr, _ := claims["sub"].(string)
	sub, _ := strconv.ParseUint(subStr, 10, 64)
	role, _ := claims["role"].(string)
	exp, _ := claims["exp"].(float64)
	iat, _ := claims["iat"].(float64)
	return Claims{
		UserID:    uint(sub),
		Role:      role,
		ExpiresAt: time.Unix(int64(exp), 0),
		IssuedAt:  time.Unix(int64(iat), 0),
	}, nil
}
