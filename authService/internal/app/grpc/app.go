package grpcapp

import (
	"fmt"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"sso/internal/config"
	authgrpc "sso/internal/grpc/auth"
	jwtlib "sso/internal/lib/jwt"
	"sso/internal/services/auth"
	"sso/internal/storage/sqlite"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(log *slog.Logger,
	cfg *config.Config,
	// mainStoragePath string,
	// sessionsStoragePath string,
	// tokenTTL time.Duration,
	// grpcPort int,
) *App {

	mainStorage, err := sqlite.New(cfg.MainStoragePath)
	if err != nil {
		panic(err)
	}

	tokenProvider := jwtlib.New(cfg.AccessTokenTTL, cfg.Secret)

	// FIXME: Нужно реализовать логику для внедрения sessionsStorage
	authService := auth.New(log, mainStorage, mainStorage, tokenProvider)

	gRPCServer := grpc.NewServer()
	authgrpc.Register(gRPCServer, authService)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       cfg.GRPC.Port,
	}
}

func (a *App) MustRun() {
	if err := a.run(); err != nil {
		panic(err)
	}
}

func (a *App) run() error {
	const op = "grpcapp.Run"

	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("grpc server is running", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s, %w", op, err)
	}
	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stoppint grpc server", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()
}
