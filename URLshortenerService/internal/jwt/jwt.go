package jwtlib

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type TokenManager struct {
	secret string
}

func New(accessTokenTTL time.Duration, refreshTokenTTL time.Duration, secret string) *TokenManager {
	return &TokenManager{
		secret: secret,
	}
}

func (t *TokenManager) ValidateTokenAndGetClaims(tokenString string) (jwt.MapClaims, error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(t.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("token validation error: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to get claims from token")
	}
	expiresAtAny, ok := claims["exp"]
	if !ok {
		return nil, errors.New("invalid fields of claims")
	}

	expiresAt, ok := expiresAtAny.(float64)
	if !ok {
		return nil, errors.New("invalid fields of claims")
	}
	if time.Now().Unix() > int64(expiresAt) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}

func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
	claims, ok := ctx.Value("claims").(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed ot get claims from context")
	}
	return claims, nil
}
