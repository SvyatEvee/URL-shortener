package update_test

import (
	"URLshortener/internal/http-server/handlers/url/update"
	"URLshortener/internal/http-server/handlers/url/update/mocks"
	"URLshortener/internal/lib/logger/handlers/slogdiscard"
	"URLshortener/internal/storage"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateHandler(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		url        string
		alias      string
		respError  string
		mockError  error
		respStatus int
	}{
		{
			name:       "Success",
			input:      `{"url": "https://google.com", "alias": "google"}`,
			url:        "https://google.com",
			alias:      "google",
			respStatus: http.StatusOK,
		},
		{
			name:       "Empty alias",
			input:      `{"url": "https://google.com"}`,
			url:        "https://google.com",
			alias:      "",
			respStatus: http.StatusOK,
		},
		{
			name:       "Empty url",
			input:      `{"alias": "some_alias"}`,
			url:        "",
			alias:      "some_alias",
			respError:  "field URL is a required field",
			respStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid URL",
			input:      `{"url": "some invalid URL", "alias": "some_alias"}`,
			url:        "some invalid URL",
			alias:      "some_alias",
			respError:  "field URL is not a valid URL",
			respStatus: http.StatusBadRequest,
		},
		{
			name:       "storage error: alias exist",
			input:      `{"url": "https://google.com", "alias": "some_alias"}`,
			url:        "https://google.com",
			alias:      "some_alias",
			respError:  "alias already exists",
			mockError:  storage.ErrAliasExist,
			respStatus: http.StatusBadRequest,
		},
		{
			name:       "storage error: url not found",
			input:      `{"url": "https://google.com", "alias": "some_alias"}`,
			url:        "https://google.com",
			alias:      "some_alias",
			respError:  "url not found",
			mockError:  storage.ErrURLNotFound,
			respStatus: http.StatusBadRequest,
		},
		{
			name:       "storage error: other error",
			input:      `{"url": "https://google.com", "alias": "some_alias"}`,
			url:        "https://google.com",
			alias:      "some_alias",
			respError:  "failed to update url",
			mockError:  errors.New("unexpected error"),
			respStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid JSON input 1",
			input:      `{"url": "https://google.com", "alias": "some_alias"`,
			url:        "https://google.com",
			alias:      "some_alias",
			respError:  "failed to decode request",
			respStatus: http.StatusBadRequest,
		},
		{
			name:       "incorrect but working JSON input 2",
			input:      `{"uRl": "https://google.com", "alIas": "some_alias"}`,
			url:        "https://google.com",
			alias:      "some_alias",
			respStatus: http.StatusOK,
		},
		{
			name:       "incorrect but working JSON input 3",
			input:      `{"url": "https://google.com", "alia": "some_alias"}`,
			url:        "https://google.com",
			alias:      "some_alias",
			respStatus: http.StatusOK,
		},
		{
			name:       "Empty request body",
			input:      "",
			respError:  "failed to decode request",
			respStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlUpdatorMock := mocks.NewURLUpdater(t)

			if tc.respError == "" || tc.mockError != nil {
				urlUpdatorMock.On("UpdateURL", tc.url, mock.AnythingOfType("string")).
					Return(tc.mockError).
					Once()
			}
			// новое тело экземпляра хендлера
			handler := update.New(slogdiscard.NewDiscardLogger(), urlUpdatorMock)

			// новое тело запроса
			req, err := http.NewRequest(http.MethodPatch, "/url", bytes.NewReader([]byte(tc.input)))
			require.NoError(t, err)

			//Для теста на Content-Type
			if tc.name == "Invalid Content-Type" {
				req.Header.Set("Content-Type", "text/plain")
			} else {
				req.Header.Set("Content-Type", "application/json")
			}

			// новое тело ответа
			rr := httptest.NewRecorder()

			// Запускаем хэндлер
			handler.ServeHTTP(rr, req)

			// Проверки после выполения хендлера
			require.Equal(t, tc.respStatus, rr.Code)

			// Считываем данные с тела ответа
			body := rr.Body.Bytes()
			var resp update.Response

			require.NoError(t, json.Unmarshal(body, &resp))
			require.Equal(t, tc.respError, resp.Response.Error)
		})
	}
}
