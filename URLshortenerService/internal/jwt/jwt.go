package jwtlib

import (
	"URLshortener/internal/domain/models"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	secret          string
}

func New(accessTokenTTL time.Duration, refreshTokenTTL time.Duration, secret string) *TokenManager {
	return &TokenManager{
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		secret:          secret,
	}
}

func (t *TokenManager) GetAccessTokenTTL() time.Duration {
	return t.accessTokenTTL
}

func (t *TokenManager) GetRefreshTokenTTL() time.Duration {
	return t.refreshTokenTTL
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

func (t *TokenManager) GenerateNewTokenPair(user *models.User) (string, string, error) {

	accessTokenString, err := t.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshTokenRandomPart, err := t.generateRefreshTokenRandomPart()
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenRandomPart, nil
}

func (t *TokenManager) generateAccessToken(user *models.User) (string, error) {
	accessToken := jwt.New(jwt.SigningMethodHS256)

	claims := accessToken.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(t.accessTokenTTL).Unix()
	claims["role"] = user.Role

	tokenString, err := accessToken.SignedString([]byte(t.secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (t *TokenManager) generateRefreshTokenRandomPart() (string, error) {
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
		return nil, errors.New("failed to get claims from context")
	}
	return claims, nil
}
