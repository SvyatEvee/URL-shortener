package deletee

import (
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

// FIXME: не нравится мне что в response тоже есть такая же структура
type request struct {
	Alias string `json:"alias" validate:"required"`
}

// хз пока надо или нет
//type Response struct {
//	resp.Response
//	Alias string `json:"alias,omitempty"`
//}

type Deleter interface {
	DeleteAlias(alias string) error
	//DeleteURL(url string) error
}

func New(log *slog.Logger, deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode body", sl.Err(err))

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

		err := deleter.DeleteAlias(req.Alias)
		switch {
		case errors.Is(err, storage.ErrAliasNotFound):
			log.Error("alias not found", slog.String("alias", req.Alias))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("alias not found"))
			return
		case err != nil:
			log.Error("failed to delete alias", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal server error"))
			return
		}

		render.Status(r, http.StatusNoContent)
	}
}
