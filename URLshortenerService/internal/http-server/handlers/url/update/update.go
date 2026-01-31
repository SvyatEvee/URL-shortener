package update

import (
	jwtlib "URLshortener/internal/jwt"
	resp "URLshortener/internal/lib/api/response"
	"URLshortener/internal/lib/logger/sl"
	"URLshortener/internal/storage"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
)

type request struct {
	ID     int64  `json:"urlId"`
	NewUrl string `json:"newUrl" validate:"required,url"`
}

type Response struct {
	ID    int64  `json:"id,omitempty"`
	Url   string `json:"url,omitempty"`
	Error string `json:"message,omitempty"`
}

//go:generate mockery --name=URLUpdater --output=./mocks
type Updater interface {
	//UpdateAlias(oldAlias string, newAlias string, userID int64) error
	UpdateAlias(id int64, newUrl string, userID int64) error
}

func New(log *slog.Logger, Updater Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.update.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if r.Header.Get("Content-Type") != "application/json" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, Response{Error: "invalid Content-Type"})
			return
		}

		// Берем из контекста данные JWT токена
		claims, err := jwtlib.GetClaimsFromContext(r.Context())
		if err != nil {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, Response{Error: "failed to get claims"})
			return
		}

		userIDAny, ok := claims["uid"]
		if !ok {
			log.Error("failed to get field uid from claims")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, Response{Error: "internal error"})
			return
		}
		userID := int64(userIDAny.(float64))

		var req request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, Response{Error: "failed to decode request"})
			return
		}
		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, Response{Error: resp.ValidationError(validateErr)})
			return
		}

		// Обновление данных в storage
		err = Updater.UpdateAlias(req.ID, req.NewUrl, userID)
		switch {
		case errors.Is(err, storage.ErrAliasNotFound):
			log.Info("alias not found", slog.Int64("url", req.ID))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, Response{Error: "alias not found"})
			return
		case err != nil:
			log.Error("failed to update alias", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, Response{Error: "failed to update alias"})
			return
		}

		log.Info("url updated")

		render.JSON(w, r, Response{
			ID:  req.ID,
			Url: req.NewUrl,
		})
	}
}
