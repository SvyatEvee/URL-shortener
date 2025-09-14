package authorization

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log/slog"
	"sso/internal/lib/logger/sl"
	"strings"
	"time"
)

type TokenValidator interface {
	ValidateTokenAndGetClaims(tokenString string) (jwt.MapClaims, error)
}

func isPublicMethod(methodName string) bool {
	publicMethod := map[string]bool{
		"/auth.Auth/Register": true,
		"/auth.Auth/Login":    true,
	}
	return publicMethod[methodName]
}

func NewJWTInterceptor(log *slog.Logger, tokenValidator TokenValidator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		const op = "interceptors.authorization.NewJWTInterceptor"
		log = log.With(
			slog.String("op", op),
			slog.String("method", info.FullMethod),
		)

		start := time.Now()
		defer func() {
			log.Debug("request processed", slog.String("duration", time.Since(start).String()))
		}()

		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Info("metadata not provided")
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}

		authHeader, ok := md["authorization"]
		if !ok || len(authHeader) == 0 {
			log.Info("authorization header missing")
			return nil, status.Error(codes.Unauthenticated, "authorization token required")
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader[0], bearerPrefix) {
			log.Info("invalid authorization format")
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}

		tokenString := strings.TrimPrefix(authHeader[0], bearerPrefix)

		claims, err := tokenValidator.ValidateTokenAndGetClaims(tokenString)
		if err != nil {
			log.Info("token validation failed", sl.Err(err))
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, "claims", claims)
		log.Debug("token validated successfully")

		return handler(ctx, req)
	}
}
