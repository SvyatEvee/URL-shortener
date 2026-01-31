package authorization

import (
	"URLshortener/internal/lib/logger/sl"
	"context"
	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"net/http"
	"strings"
)

const (
	claimsKey string = "claims"
)

type TokenValidator interface {
	ValidateTokenAndGetClaims(tokenString string) (jwt.MapClaims, error)
}

func New(log *slog.Logger, tokenValidator TokenValidator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		log.Info("authorization middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{
					"error": "Authorization header is required",
				})
				return
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				log.Info("invalid authorization format")
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{
					"error": "Invalid Authorization header format",
				})
				return
			}

			tokenString := strings.TrimPrefix(authHeader, bearerPrefix)

			claims, err := tokenValidator.ValidateTokenAndGetClaims(tokenString)
			if err != nil {
				log.Info("token validation failed", sl.Err(err))
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{
					"error": "Invalid token",
				})
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			log.Debug("token validated successfully")
			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}
