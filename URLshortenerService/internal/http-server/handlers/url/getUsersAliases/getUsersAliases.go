package getUsersAliases

import (
	"URLshortener/internal/domain/models"
	jwtlib "URLshortener/internal/jwt"
	"URLshortener/internal/storage"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Error string `json:"message,omitempty"`
}

//go:generate mockery --name=URLSaver --output=./mocks
type UsersDataProvider interface {
	GetUserUrls(userID int64) ([]models.AliasNote, error)
}

func New(log *slog.Logger, usersDataProvider UsersDataProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.getUsersAliases.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Берем из контекста данные JWT токена
		claims, err := jwtlib.GetClaimsFromContext(r.Context())
		if err != nil {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Error: "failed to get claims"})
			return
		}

		userIDAny, ok := claims["uid"]
		if !ok {
			log.Error("failed to get field uid from claims")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Error: "internal error"})
			return
		}
		userID := int64(userIDAny.(float64))

		// Запись в storage
		usersAliases, err := usersDataProvider.GetUserUrls(userID)

		if err != nil && !errors.Is(err, storage.ErrAliasNotFound) {
			log.Error("failed to get aliases", slog.String("error", err.Error()))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Error: "failed to add alias"})
			return
		}

		if usersAliases == nil {
			usersAliases = []models.AliasNote{}
		}

		render.JSON(w, r, usersAliases)
	}
}
