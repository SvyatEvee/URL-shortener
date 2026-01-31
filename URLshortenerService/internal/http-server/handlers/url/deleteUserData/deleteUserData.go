package deleteUserData

import (
	jwtlib "URLshortener/internal/jwt"
	resp "URLshortener/internal/lib/api/response"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type Deleter interface {
	DeleteUserData(userID int64) error
}

func New(log *slog.Logger, deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.deleteUserData.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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

		roleAny, ok := claims["role"]
		if !ok {
			log.Error("failed to get field role from claims")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}
		role := roleAny.(string)

		if !(role == "service" || role == "admin") {
			log.Error("role is not service or admin")
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Error("Forbidden"))
			return
		}

		if err := deleter.DeleteUserData(userID); err != nil {
			log.Error("failed to delete user's data")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, nil)
	}
}
