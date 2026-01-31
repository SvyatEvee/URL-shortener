package grpcapp

import (
	"context"
	"errors"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"net"
	"net/http"
	"sso/gen/go/sso"
	"sso/internal/config"
	authgrpc "sso/internal/grpc/auth"
	"sso/internal/grpc/interceptors/authorization"
	"sso/internal/http/urlServiceSender"
	jwtlib "sso/internal/lib/jwt"
	"sso/internal/services/auth"
	"sso/internal/storage/sql"
	"time"
)

type App struct {
	log            *slog.Logger
	gRPCServer     *grpc.Server
	port           int
	gatewayServer  *http.Server
	gatewayEnabled bool
}

func New(log *slog.Logger, cfg *config.Config) *App {

	mainStorage, err := sql.New(cfg.MainStorageDBDriver, cfg.MainStorageConnString)
	if err != nil {
		panic(err)
	}

	sessionStorage, err := sql.New(cfg.SessionsStorageDBDriver, cfg.SessionsStorageConnString)
	if err != nil {
		panic(err)
	}

	tokenManager := jwtlib.New(cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.Secret)

	urlServiceManager := urlServiceSender.New(log, fmt.Sprintf("%s:%d", cfg.UrlService.Host, cfg.UrlService.Port))

	authService := auth.New(log, mainStorage, tokenManager, sessionStorage, urlServiceManager)

	gRPCServer := grpc.NewServer(
		grpc.UnaryInterceptor(authorization.NewJWTInterceptor(log, tokenManager)),
	)

	authgrpc.Register(gRPCServer, authService)

	var gatewaySrv *http.Server
	gatewayEnabled := false
	if cfg.Gateway.Enabled {
		gatewayEnabled = true

		ctx := context.Background()

		// спец. мультиплексор grpc-gateway
		mux := runtime.NewServeMux()

		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}

		err := sso.RegisterAuthHandlerFromEndpoint(
			ctx,
			mux,
			fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port),
			opts,
		)
		if err != nil {
			panic(err)
		}

		gatewaySrv = &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Gateway.Port),
			Handler:      mux,
			ReadTimeout:  cfg.Gateway.Timeout,
			WriteTimeout: cfg.Gateway.Timeout,
			IdleTimeout:  cfg.Gateway.IdleTimeout,
		}

	}

	return &App{
		log:            log,
		gRPCServer:     gRPCServer,
		port:           cfg.GRPC.Port,
		gatewayServer:  gatewaySrv,
		gatewayEnabled: gatewayEnabled,
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
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	gRPCServerError := make(chan error)
	go func() {
		defer close(gRPCServerError)
		log.Info("grpc server is running", slog.String("addr", l.Addr().String()))

		if err := a.gRPCServer.Serve(l); err != nil {
			gRPCServerError <- fmt.Errorf("%s, %w", op, err)
			return
		}
		gRPCServerError <- nil

	}()

	var GatewayServerError chan error
	if a.gatewayEnabled {
		GatewayServerError = make(chan error)
		go func() {
			defer close(GatewayServerError)
			log.Info("grpc gateway server is running", slog.String("Addr", a.gatewayServer.Addr))

			err := a.gatewayServer.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				GatewayServerError <- fmt.Errorf("%s, %w", op, err)
				return
			}
			GatewayServerError <- nil

		}()
	}

	select {
	case err = <-gRPCServerError:
		return err
	case err = <-GatewayServerError:
		return err
	}
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stopping grpc server", slog.Int("port", a.port))

	if a.gatewayEnabled {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := a.gatewayServer.Shutdown(shutdownCtx); err != nil {
			a.log.Error("failed to Shutdown gateway", slog.String("error", err.Error()))
		} else {
			a.log.Info("gateway server is stopped")
		}
	}
	a.gRPCServer.GracefulStop()
}
