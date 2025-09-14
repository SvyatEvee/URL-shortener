package jwtlib

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"sso/internal/domain/models"
	"time"
)

type TokenManager struct {
	tokenTTL time.Duration
	secret   string
}

func New(tokenTTL time.Duration, secret string) *TokenManager {
	return &TokenManager{
		tokenTTL: tokenTTL,
		secret:   secret,
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
		return nil, errors.New("failed ot get claims from token")
	}

	return claims, nil
}

func (t *TokenManager) GenerateNewTokenPair(user models.User) (string, string, error) {

	accessTokenString, err := t.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := t.generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshToken, nil
}

func (t *TokenManager) generateAccessToken(user models.User) (string, error) {
	accessToken := jwt.New(jwt.SigningMethodHS256)

	claims := accessToken.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(t.tokenTTL).Unix()
	claims["role"] = user.Role

	tokenString, err := accessToken.SignedString([]byte(t.secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (t *TokenManager) generateRefreshToken() (string, error) {
	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	refreshToken := base64.URLEncoding.EncodeToString(tokenBytes)
	return refreshToken, nil
}

func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
	claims, ok := ctx.Value("claims").(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed ot get claims from context")
	}
	return claims, nil
}
