package save

import (
	jwtlib "URLshortener/internal/jwt"
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

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	Id    int64  `json:"id,omitempty"`
	Url   string `json:"url,omitempty"`
	Alias string `json:"alias,omitempty"`
	Error string `json:"message,omitempty"`
}

// TODO: move to config
const AliasLength = 6

//go:generate mockery --name=URLSaver --output=./mocks
type URLSaver interface {
	SaveURL(urlToSave string, alias string, userID int64) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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

		//roleAny, ok := claims["role"]
		//if !ok {
		//	log.Error("failed to get field role from claims")
		//	render.Status(r, http.StatusInternalServerError)
		//	render.JSON(w, r, resp.Error("internal error"))
		//	return
		//}
		//role := roleAny.(string)

		// Считывание из JSON
		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, Response{Error: "failed to decode request"})
			return
		}

		log.Info("request body decoded", slog.Any("request", req))
		// Валидация считанной структуры
		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, Response{Error: resp.ValidationError(validateErr)})
			return
		}

		// Проверка на наличие alias в запросе
		// TODO: проверка на ошибку одинаковых рандомных имен
		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(AliasLength)
		}
		// Запись в storage
		id, err := urlSaver.SaveURL(req.URL, alias, userID)
		switch {
		case errors.Is(err, storage.ErrAliasExist):
			log.Info("alias already exists", slog.String("url", req.URL))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, Response{Error: "Такой алиас уже существует"})
			return
		case err != nil:
			log.Error("failed to add alias", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, Response{Error: "failed to add alias"})
			return
		}

		log.Info("url added", slog.Int64("id", id))

		render.JSON(w, r, Response{
			Id:    id,
			Url:   req.URL,
			Alias: alias,
		})
	}
}
