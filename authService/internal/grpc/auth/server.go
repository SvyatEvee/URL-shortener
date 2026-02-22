package auth

import (
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	ssov1 "sso/gen/go/sso"
	"sso/internal/lib/api"
	"sso/internal/services/auth"
)

type Auth interface {
	Login(ctx context.Context,
		email string,
		password string) (string, string, error)
	RegisterNewUser(ctx context.Context,
		email string,
		password string) (int64, error)
	GetNewRefreshToken(ctx context.Context,
		refreshToken string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	DeleteUserByID(ctx context.Context, userID int64) error
	DeleteUserByEmail(ctx context.Context, userEmail string) error
}

type serverAPI struct {
	ssov1.UnimplementedAuthServer
	auth Auth
}

func Register(gRPC *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPC, &serverAPI{auth: auth})
}

func (s *serverAPI) DeleteUserByID(ctx context.Context, req *ssov1.DeleteUserByIDRequest) (*ssov1.DeleteUserByIDResponse, error) {

	if err := s.auth.DeleteUserByID(ctx, req.GetUserId()); err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			return nil, status.Error(codes.PermissionDenied, "invalid credentials")
		case errors.Is(err, auth.ErrUserNotFound):
			return nil, status.Error(codes.InvalidArgument, "user non-exists")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	return &ssov1.DeleteUserByIDResponse{Success: true}, nil
}

func (s *serverAPI) DeleteUserByEmail(ctx context.Context, req *ssov1.DeleteUserByEmailRequest) (*ssov1.DeleteUserByEmailResponse, error) {

	if err := validateEmail(req); err != nil {
		return nil, err
	}

	if err := s.auth.DeleteUserByEmail(ctx, req.GetEmail()); err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			return nil, status.Error(codes.PermissionDenied, "invalid credentials")
		case errors.Is(err, auth.ErrUserNotFound):
			return nil, status.Error(codes.InvalidArgument, "user non-exists")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	return &ssov1.DeleteUserByEmailResponse{Success: true}, nil
}

func (s *serverAPI) Logout(ctx context.Context, req *ssov1.LogoutRequest) (*ssov1.LogoutResponse, error) {

	if err := s.auth.Logout(ctx, req.GetRefreshToken()); err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrInvalidRefreshToken):
			return nil, status.Error(codes.PermissionDenied, "Доступ запрещен")
		case errors.Is(err, auth.ErrSessionNotFound):
			return nil, status.Error(codes.InvalidArgument, "Такой сессии не существует")
		default:
			return nil, status.Error(codes.Internal, "Внутрення ошибка сервера")
		}
	}

	return &ssov1.LogoutResponse{Success: true}, nil
}

func (s *serverAPI) GetNewRefreshToken(ctx context.Context, req *ssov1.GetNewRefreshTokenRequest) (*ssov1.GetNewRefreshTokenResponse, error) {

	accessToken, refreshToken, err := s.auth.GetNewRefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrSessionExpired):
			return nil, status.Error(codes.Unauthenticated, "session expired")
		case errors.Is(err, auth.ErrSessionNotFound):
			return nil, status.Error(codes.InvalidArgument, "a non-existent session")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.GetNewRefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *serverAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {

	if err := validateLogin(req); err != nil {
		return nil, err
	}

	accessToken, refreshToken, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "Неверный логин или пароль")
		}

		return nil, status.Error(codes.Internal, "Внутренняя ошибка сервера")
	}

	return &ssov1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *serverAPI) Register(ctx context.Context, req *ssov1.RegisterRequest) (*ssov1.RegisterResponse, error) {
	if err := validateRegister(req); err != nil {
		return nil, err
	}

	userID, err := s.auth.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "Пользователь с таким email уже существует")
		}

		return nil, status.Error(codes.Internal, "Внутренняя ошибка сервера")
	}

	return &ssov1.RegisterResponse{
		UserId: userID,
	}, nil
}

func validateLogin(req *ssov1.LoginRequest) error {

	type loginRequestValidate struct {
		Email    string `validate:"required"`
		Password string `validate:"required"`
	}

	toValidate := loginRequestValidate{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}

	if err := validator.New().Struct(toValidate); err != nil {
		var validateErr validator.ValidationErrors
		if errors.As(err, &validateErr) {
			return status.Error(codes.InvalidArgument, api.ValidationError(validateErr))
		}
		return status.Error(codes.InvalidArgument, "login request validation is failed")
	}

	return nil
}

func validateRegister(req *ssov1.RegisterRequest) error {
	type registerRequestValidate struct {
		Email    string `validate:"required"`
		Password string `validate:"required"`
	}

	toValidate := registerRequestValidate{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}

	if err := validator.New().Struct(toValidate); err != nil {
		var validateErr validator.ValidationErrors
		if errors.As(err, &validateErr) {
			return status.Error(codes.InvalidArgument, api.ValidationError(validateErr))
		}
		return status.Error(codes.InvalidArgument, "register request validation is failed")
	}

	return nil
}

func validateEmail(req *ssov1.DeleteUserByEmailRequest) error {
	validate := validator.New()

	email := req.GetEmail()
	if err := validate.Var(email, "required"); err != nil {
		var validateErr validator.ValidationErrors
		if errors.As(err, &validateErr) {
			return status.Error(codes.InvalidArgument, api.ValidationError(validateErr))
		}
		return status.Error(codes.InvalidArgument, "register request validation is failed")
	}
	return nil
}
