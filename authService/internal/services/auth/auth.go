package auth

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage"
	"time"
)

type Auth struct {
	log          *slog.Logger
	usrSaver     UserSaver
	usrProvider  UserProvider
	tokenManager TokenManager
}

type UserSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte) (int64, error)
}

type UserProvider interface {
	GetUser(ctx context.Context, email string) (models.User, error)
}

type TokenManager interface {
	GenerateNewTokenPair(user models.User) (string, string, error)
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAppID       = errors.New("invalid app id")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
)

// FIXME: добавить интрефейс sessionStorage
// New returns a new instance of the Auth service
func New(log *slog.Logger, userSaver UserSaver, userProvider UserProvider, tokenManager TokenManager) *Auth {
	return &Auth{
		usrSaver:     userSaver,
		usrProvider:  userProvider,
		log:          log,
		tokenManager: tokenManager,
	}
}

func (a *Auth) Login(ctx context.Context, email string, password string) (string, string, error) {

	const op = "auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email))

	log.Info("attempting to user login")

	user, err := a.usrProvider.GetUser(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", sl.Err(err))

			return "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		log.Error("failed to get user", sl.Err(err))

		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Info("invalid credentials", sl.Err(err))

		return "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	//app, err := a.appProvider.App(ctx, appID)
	//if err != nil {
	//	return "", fmt.Errorf("%s: %w", op, err)
	//}

	log.Info("user logged in successfully")

	accessToken, refreshToken, err := a.tokenManager.GenerateNewTokenPair(user)
	if err != nil {
		log.Error("failed to generate token", sl.Err(err))

		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	return accessToken, refreshToken, nil
}

func (a *Auth) RegisterNewUser(ctx context.Context, email string, password string) (int64, error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email))

	start := time.Now()
	defer func() {
		log.Debug("request processed", slog.String("duration", time.Since(start).String()))
	}()

	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.usrSaver.SaveUser(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exists", sl.Err(err))

			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered")
	return id, nil
}

//func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
//	const op = "auth.IsAdmin"
//
//	log := a.log.With(
//		slog.String("op", op),
//		slog.Int64("user_id", userID))
//
//	log.Info("checking if user is admin")
//
//	isAdmin, err := a.usrProvider.IsAdmin(ctx, userID)
//	if err != nil {
//		if errors.Is(err, storage.ErrUserNotFound) {
//			log.Warn("user not found", sl.Err(err))
//
//			return false, fmt.Errorf("%s: %w", op, ErrUserNotFound)
//		}
//
//		return false, fmt.Errorf("%s: %w", op, err)
//	}
//
//	log.Info("checked if user is admin", slog.Bool("is_admin", isAdmin))
//
//	return isAdmin, nil
//}
