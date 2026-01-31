package deletee

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
	ID int64 `json:"urlId"`
}

type response struct {
	Error string `json:"message,omitempty"`
}

type Deleter interface {
	DeleteAlias(id int64, userID int64) error
	//DeleteURL(url string) error
}

func New(log *slog.Logger, deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Берем из контекста данные JWT токена
		claims, err := jwtlib.GetClaimsFromContext(r.Context())
		if err != nil {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response{Error: "failed to get claims"})
			return
		}

		userIDAny, ok := claims["uid"]
		if !ok {
			log.Error("failed to get field uid from claims")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response{Error: "internal error"})
			return
		}
		userID := int64(userIDAny.(float64))

		//byteData, err := io.ReadAll(r.Body)
		//strData := string(byteData)
		//byteData = []byte(strData)

		var req request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode body", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response{Error: "failed to decode request"})
			return
		}

		log.Info("request body decoded", slog.Any("request", req))
		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validateErr))
			render.JSON(w, r, response{Error: resp.ValidationError(validateErr)})
			return
		}

		err = deleter.DeleteAlias(req.ID, userID)
		switch {
		case errors.Is(err, storage.ErrAliasNotFound):
			log.Error("alias not found", slog.Int64("aliasID", req.ID))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, response{Error: "alias not found"})
			return
		case err != nil:
			log.Error("failed to delete alias", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response{Error: "internal server error"})
			return
		}

		render.NoContent(w, r)
	}
}
