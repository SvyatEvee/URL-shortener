package redirect

import (
	jwtlib "URLshortener/internal/jwt"
	resp "URLshortener/internal/lib/api/response"
	"URLshortener/internal/lib/logger/sl"
	"URLshortener/internal/storage"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type URLGetter interface {
	GetURL(alias string, userID int64) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		claims, err := jwtlib.GetClaimsFromContext(r.Context())
		if err != nil {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to get claims"))
			return
		}

		userIDAny, ok := claims["uid"]
		if !ok {
			log.Error("failed to get field uid from claims")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}
		userID := int64(userIDAny.(float64))

		resURL, err := urlGetter.GetURL(alias, userID)
		switch {
		case errors.Is(err, storage.ErrAliasNotFound):
			log.Info("alias not found", slog.String("alias", alias))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("not found"))
			return
		case err != nil:
			log.Error("failed to get url", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		log.Info("got url", slog.String("url", resURL))

		//http.Redirect(w, r, resURL, http.StatusFound)
		w.Header().Set("Location", resURL)
		w.WriteHeader(http.StatusOK)
		return
	}
}
