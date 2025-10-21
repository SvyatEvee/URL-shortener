package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"sso/internal/domain/models"
	jwtlib "sso/internal/lib/jwt"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage"
	"time"
)

type Auth struct {
	log            *slog.Logger
	userManager    UserManager
	tokenManager   TokenManager
	sessionManager SessionManager
}

type RefreshTokenPayload struct {
	SessionID  int64  `json:"session_id"`
	RandomPart string `json:"random_part"`
}

type UserManager interface {
	SaveUser(ctx context.Context, email string, passHash []byte) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	GetUserByID(ctx context.Context, userID int64) (models.User, error)
	DeleteUser(ctx context.Context, userID int64) error
}

type TokenManager interface {
	GenerateNewTokenPair(user *models.User) (string, string, error)
	GetRefreshTokenTTL() time.Duration
}

type SessionManager interface {
	SaveSession(ctx context.Context, session models.Session) (int64, error)
	GetSession(ctx context.Context, sessionID int64) (*models.Session, error)
	DeleteSession(ctx context.Context, sessionID int64) error
	UpdateSession(ctx context.Context, newSession *models.Session) error
	DeleteAllUserSessions(ctx context.Context, userID int64) (int64, error)
}

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserExists          = errors.New("user already exists")
	ErrSessionNotFound     = errors.New("session not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrSessionExpired      = errors.New("session expired")
)

// New returns a new instance of the Auth service
func New(log *slog.Logger, userManager UserManager, tokenManager TokenManager, sessionManager SessionManager) *Auth {
	return &Auth{
		userManager:    userManager,
		log:            log,
		tokenManager:   tokenManager,
		sessionManager: sessionManager,
	}
}

func (a *Auth) GetNewRefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	const op = "auth.GetNewRefreshToken"

	log := a.log.With(
		slog.String("op", op))

	start := time.Now()
	defer func() {
		log.Debug("request processed", slog.String("duration", time.Since(start).String()))
	}()

	tokenJSON, err := base64.URLEncoding.DecodeString(refreshToken)
	if err != nil {
		log.Error("failed to decode refreshToken to JSON", slog.String("err", err.Error()))
		return "", "", err
	}

	var payload RefreshTokenPayload
	if err := json.Unmarshal(tokenJSON, &payload); err != nil {
		log.Error("failed to unmarshal json", slog.String("err", err.Error()))
		return "", "", err
	}

	session, err := a.sessionManager.GetSession(ctx, payload.SessionID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrSessionNotFound):
			log.Warn("session not found", slog.String("error", err.Error()))
			return "", "", ErrSessionNotFound
		default:
			log.Error("failed to get session", slog.String("error", err.Error()))
			return "", "", err
		}
	}

	err = bcrypt.CompareHashAndPassword(session.RefreshTokenRandomPartHash, []byte(payload.RandomPart))
	if err != nil {
		log.Info("invalid refresh token", slog.String("error", err.Error()))

		return "", "", ErrInvalidRefreshToken
	}

	if time.Now().Unix() > session.ExpiresAt {

		log.Info("session expired",
			slog.Int64("session_id", session.ID))

		err := a.sessionManager.DeleteSession(ctx, session.ID)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrSessionNotFound):
				log.Warn("not found expired session", slog.String("error", err.Error()))
				return "", "", err
			default:
				log.Error("failed to delete expired session", slog.String("error", err.Error()))
				return "", "", err
			}
		}

		return "", "", ErrSessionExpired
	}

	user, err := a.userManager.GetUserByID(ctx, session.UserID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			log.Error("There is a session for a non-existent user", slog.String("error", err.Error()))
			return "", "", ErrUserNotFound
		default:
			log.Error("failed to get user", slog.String("error", err.Error()))
			return "", "", err
		}
	}

	accessToken, refreshTokenRandPart, err := a.tokenManager.GenerateNewTokenPair(&user)
	if err != nil {
		log.Error("failed to generate new token pair", slog.String("error", err.Error()))

		return "", "", err
	}

	refreshTokenRandomPartHash, err := bcrypt.GenerateFromPassword([]byte(refreshTokenRandPart), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate refreshTokenRandPart hash", slog.String("error", err.Error()))
		return "", "", err
	}

	newSession := &models.Session{
		ID:                         session.ID,
		UserID:                     session.UserID,
		RefreshTokenRandomPartHash: refreshTokenRandomPartHash,
		CreatedAt:                  time.Now().Unix(),
		ExpiresAt:                  time.Now().Add(a.tokenManager.GetRefreshTokenTTL()).Unix(),
	}

	if err := a.sessionManager.UpdateSession(ctx, newSession); err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			log.Error("updating a non-existent session", slog.String("error", err.Error()))
			return "", "", err
		}

		log.Error("failed to update session", slog.String("error", err.Error()))
		return "", "", err
	}

	newPayload := &RefreshTokenPayload{
		SessionID:  newSession.ID,
		RandomPart: refreshTokenRandPart,
	}

	newTokenJSON, err := json.Marshal(newPayload)
	if err != nil {
		log.Error("failed to generate json", slog.String("error", err.Error()))
		return "", "", err
	}

	newRefreshToken := base64.URLEncoding.EncodeToString(newTokenJSON)

	log.Info("a pair of token have been updated",
		slog.Int64("user_id", newSession.UserID),
		slog.Int64("session_id", newSession.ID),
	)
	return accessToken, newRefreshToken, nil
}

func (a *Auth) Login(ctx context.Context, email string, password string) (string, string, error) {

	const op = "auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email))

	start := time.Now()
	defer func() {
		log.Debug("request processed", slog.String("duration", time.Since(start).String()))
	}()

	log.Info("attempting to user login")

	user, err := a.userManager.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", sl.Err(err))

			return "", "", ErrInvalidCredentials
		}

		log.Error("failed to get user", sl.Err(err))

		return "", "", err
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Info("invalid credentials", sl.Err(err))

		return "", "", ErrInvalidCredentials
	}

	log.Info("user logged in successfully")

	accessToken, refreshTokenRandomPart, err := a.tokenManager.GenerateNewTokenPair(&user)
	if err != nil {
		log.Error("failed to generate new token pair", sl.Err(err))

		return "", "", err
	}

	refreshTokenRandomPartHash, err := bcrypt.GenerateFromPassword([]byte(refreshTokenRandomPart), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate refreshTokenRandPart hash", slog.String("error", err.Error()))
		return "", "", err
	}

	session := models.Session{
		UserID:                     user.ID,
		RefreshTokenRandomPartHash: refreshTokenRandomPartHash,
		CreatedAt:                  time.Now().Unix(),
		ExpiresAt:                  time.Now().Add(a.tokenManager.GetRefreshTokenTTL()).Unix(),
	}

	sessionID, err := a.sessionManager.SaveSession(ctx, session)
	if err != nil {
		log.Error("failed to save session", sl.Err(err))
		return "", "", err
	}

	payload := RefreshTokenPayload{
		SessionID:  sessionID,
		RandomPart: refreshTokenRandomPart,
	}

	tokenJSON, err := json.Marshal(payload)
	if err != nil {
		log.Error("failed to generate tokenJSON", slog.String("error:", err.Error()))
		return "", "", err
	}

	refreshToken := base64.URLEncoding.EncodeToString(tokenJSON)

	log.Info("user is logged in", slog.Int64("session_id", sessionID))
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

		return 0, err
	}

	id, err := a.userManager.SaveUser(ctx, email, passHash)
	if err != nil {
		err = fmt.Errorf("%s: %w", op, ErrUserExists)
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exists", sl.Err(err))

			return 0, err
		}
		log.Error("failed to save user", sl.Err(err))

		return 0, err
	}

	log.Info("user registered")
	return id, nil
}

func (a *Auth) Logout(ctx context.Context, refreshToken string) error {
	const op = "auth.Logout"

	log := a.log.With(
		slog.String("op", op))
	log.Debug("logout started")

	claims, err := jwtlib.GetClaimsFromContext(ctx)
	if err != nil {
		log.Error("failed to get claims from context", slog.String("error", err.Error()))
		return err
	}

	userIDFromJWT, ok := claims["uid"]
	if !ok {
		log.Error("failed to get uid from claims")
		return errors.New("failed to get uid from claims")
	}

	tokenJSON, err := base64.URLEncoding.DecodeString(refreshToken)
	if err != nil {
		log.Error("failed to decode refreshToken to JSON", slog.String("err", err.Error()))
		return err
	}

	var payload RefreshTokenPayload
	if err := json.Unmarshal(tokenJSON, &payload); err != nil {
		log.Error("failed to unmarshal json", slog.String("err", err.Error()))
		return err
	}

	log = log.With(
		slog.String("op", op),
		slog.Int64("session_id", payload.SessionID))
	log.Debug("session id added")

	session, err := a.sessionManager.GetSession(ctx, payload.SessionID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrSessionNotFound):
			log.Warn("session not found", slog.String("error", err.Error()))
			return ErrSessionNotFound
		default:
			log.Error("failed to get session", slog.String("error", err.Error()))
			return err
		}
	}

	if session.UserID != int64(userIDFromJWT.(float64)) {
		log.Info("session.UserID not equal userID from JWT",
			slog.Int64("session.UserID", session.UserID),
			slog.Int64("userIDFromJWT", int64(userIDFromJWT.(float64))))
		return ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword(session.RefreshTokenRandomPartHash, []byte(payload.RandomPart))
	if err != nil {
		log.Info("invalid refresh token", slog.String("error", err.Error()))
		return ErrInvalidRefreshToken
	}

	err = a.sessionManager.DeleteSession(ctx, session.ID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrSessionNotFound):
			log.Error("session not found", slog.String("error", err.Error()))
			return err
		default:
			log.Error("failed to delete session", slog.String("error", err.Error()))
			return err
		}
	}

	log.Info("user has successfully logged out")
	return nil
}

func (a *Auth) DeleteUserByID(ctx context.Context, userID int64) error {
	const op = "auth.DeleteUserByID"

	log := a.log.With(
		slog.String("op", op))

	start := time.Now()
	defer func() {
		log.Debug("request processed", slog.String("duration", time.Since(start).String()))
	}()

	claims, err := jwtlib.GetClaimsFromContext(ctx)
	if err != nil {
		log.Error("failed to get claims from context", slog.String("error", err.Error()))
		return err
	}

	floatUserID, ok := claims["uid"]
	if !ok {
		log.Error("failed to get uid from claims")
		return errors.New("failed to get uid from claims")
	}

	role, ok := claims["role"]
	if !ok {
		log.Error("failed to get role from claims")
		return errors.New("failed to get role from claims")
	}

	if role != "admin" && userID != int64(floatUserID.(float64)) {
		log.Info("invalid credentials")
		return ErrInvalidCredentials
	}

	log = a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID))

	_, err = a.userManager.GetUserByID(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			log.Error("attempt to delete a non-existent user", slog.String("error", err.Error()))
			return ErrUserNotFound
		default:
			log.Error("couldn't access the user for deletion", slog.String("error", err.Error()))
			return err
		}
	}

	deleteNumber, err := a.sessionManager.DeleteAllUserSessions(ctx, userID)
	if err != nil {
		log.Error("failed to delete users's sessions", slog.String("error", err.Error()))
		return err
	}

	err = a.userManager.DeleteUser(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			log.Error("the existing user is missing", slog.String("error", err.Error()))
			return err
		default:
			log.Error("failed to delete user", slog.String("error", err.Error()))
			return err
		}
	}

	log.Info("user has been successfully deleted",
		slog.Int64("deleted sessions", deleteNumber))
	return nil
}

func (a *Auth) DeleteUserByEmail(ctx context.Context, userEmail string) error {
	const op = "auth.DeleteUserByEmail"

	log := a.log.With(
		slog.String("op", op))

	start := time.Now()
	defer func() {
		log.Debug("request processed", slog.String("duration", time.Since(start).String()))
	}()

	claims, err := jwtlib.GetClaimsFromContext(ctx)
	if err != nil {
		log.Error("failed to get claims from context", slog.String("error", err.Error()))
		return err
	}

	floatUserID, ok := claims["uid"]
	if !ok {
		log.Error("failed to get uid from claims")
		return errors.New("failed to get uid from claims")
	}

	ourRole, ok := claims["role"]
	if !ok {
		log.Error("failed to get role from claims")
		return errors.New("failed to get role from claims")
	}

	user, err := a.userManager.GetUserByEmail(ctx, userEmail)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			log.Error("attempt to delete a non-existent user", slog.String("error", err.Error()))
			return ErrUserNotFound
		default:
			log.Error("couldn't access the user for deletion", slog.String("error", err.Error()))
			return err
		}
	}

	if ourRole != "admin" && user.ID != int64(floatUserID.(float64)) {
		log.Info("invalid credentials")
		return ErrInvalidCredentials
	}

	log = a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", user.ID))

	deleteNumber, err := a.sessionManager.DeleteAllUserSessions(ctx, user.ID)
	if err != nil {
		log.Error("failed to delete users's sessions", slog.String("error", err.Error()))
		return err
	}

	err = a.userManager.DeleteUser(ctx, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUserNotFound):
			log.Error("the existing user is missing", slog.String("error", err.Error()))
			return err
		default:
			log.Error("failed to delete user", slog.String("error", err.Error()))
			return err
		}
	}

	log.Info("user has been successfully deleted",
		slog.Int64("deleted sessions", deleteNumber))
	return nil
}
