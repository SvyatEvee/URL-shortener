package auth

import (
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	ssov1 "github.com/svyatevee/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sso/internal/lib/api"
	"sso/internal/services/auth"
)

const (
	emptyValue = 0
)

type Auth interface {
	Login(ctx context.Context,
		email string,
		password string) (string, string, error)
	RegisterNewUser(ctx context.Context,
		email string,
		password string) (int64, error)
}

type serverAPI struct {
	ssov1.UnimplementedAuthServer
	auth Auth
}

func Register(gRPC *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPC, &serverAPI{auth: auth})
}

func (s *serverAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {

	if err := validateLogin(req); err != nil {
		return nil, err
	}

	accessToken, refreshToken, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid email or password")
		}

		return nil, status.Error(codes.Internal, "internal error")
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
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.RegisterResponse{
		UserId: userID,
	}, nil
}

func validateLogin(req *ssov1.LoginRequest) error {

	type loginRequestValidate struct {
		Email    string `validate:"required,email"`
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
		Email    string `validate:"required,email"`
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
