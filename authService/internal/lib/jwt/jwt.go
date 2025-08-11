package jwtlib

import (
	"crypto/rand"
	"encoding/base64"
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

func (t *TokenManager) NewTokenPair(user models.User) (string, string, error) {

	accessTokenString, err := t.createAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := t.createAccessToken(user)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshToken, nil
}

func (t *TokenManager) createAccessToken(user models.User) (string, error) {
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

// TODO: покрыть тестами
//func NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
//	token := jwt.New(jwt.SigningMethodHS256)
//
//	claims := token.Claims.(jwt.MapClaims)
//	claims["uid"] = user.ID
//	claims["email"] = user.Email
//	claims["exp"] = time.Now().Add(duration).Unix()
//
//	tokenString, err := token.SignedString([]byte(app.Secret))
//	if err != nil {
//		return "", err
//	}
//
//	return tokenString, nil
//}
