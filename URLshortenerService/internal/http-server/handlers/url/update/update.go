package update

import (
	"URLshortener/internal/http-server/handlers/url/save"
	resp "URLshortener/internal/lib/api/response"
	"URLshortener/internal/lib/logger/sl"
	"URLshortener/internal/lib/random"
	"URLshortener/internal/storage"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
)

type request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

//go:generate mockery --name=URLUpdater --output=./mocks
type URLUpdater interface {
	UpdateURL(string, string) error
}

func New(log *slog.Logger, URLUpdater URLUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.update.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if r.Header.Get("Content-Type") != "application/json" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid Content-Type"))
			return
		}

		var req request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to decode request"))
			return
		}
		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validateErr))
			return
		}

		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(save.AliasLength)
		}

		// Обновление данных в storage
		err := URLUpdater.UpdateURL(req.URL, alias)
		switch {
		case errors.Is(err, storage.ErrAliasExist):
			log.Info("alias already exists", slog.String("url", req.URL))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("alias already exists"))
			return
		case errors.Is(err, storage.ErrURLNotFound):
			log.Info("url not found", slog.String("url", req.URL))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("url not found"))
			return
		case err != nil:
			log.Error("failed to update url", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to update url"))
			return
		}

		log.Info("url updated")

		render.JSON(w, r, Response{
			Response: resp.OK(),
			Alias:    alias,
		})
	}
}
