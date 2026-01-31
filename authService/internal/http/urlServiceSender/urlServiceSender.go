package urlServiceSender

import (
	"fmt"
	"log/slog"
	"net/http"
	"sso/internal/services/auth"
)

type UrlServiceSender struct {
	log            *slog.Logger
	urlServiceAddr string
}

func New(log *slog.Logger, urlServiceAddr string) *UrlServiceSender {
	return &UrlServiceSender{
		log:            log,
		urlServiceAddr: urlServiceAddr,
	}
}

func (u *UrlServiceSender) DeleteUser(serviceToken string) error {
	const op = "http.urlServiceSender.DeleteUser"

	log := u.log.With(
		slog.String("op", op))

	client := &http.Client{}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s/admin", u.urlServiceAddr), nil)
	if err != nil {
		log.Error("failed to create new request", slog.String("error", err.Error()))
		return auth.ErrInternalError
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", serviceToken))

	resp, err := client.Do(req)
	if err != nil {
		log.Error("error when trying to send a request", slog.String("error", err.Error()))
		return auth.ErrInternalError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		switch resp.StatusCode {
		case http.StatusForbidden:
			return auth.ErrInvalidCredentials
		default:
			return auth.ErrInternalError
		}
	}

	return nil
}
