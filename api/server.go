package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"bundle-server/api/app/router"
	"bundle-server/database"

	"go.uber.org/zap"
)

const (
	ReadHeaderTimeout = 5
	IdleTimeout       = 30
)

type Server struct {
	*http.Server
}

func NewServer(port string, logger *zap.Logger) (*Server, error) {
	logger.Debug("Configuring server...")
	api, err := router.New(logger)
	if err != nil {
		return nil, fmt.Errorf("[%s]: %w", "ROUTER_INIT_FAIL", err)
	}

	srv := http.Server{
		Addr:              port,
		Handler:           api,
		ReadHeaderTimeout: time.Duration(ReadHeaderTimeout) * time.Second,
		IdleTimeout:       time.Duration(IdleTimeout) * time.Second,
	}

	return &Server{&srv}, nil
}

func (srv *Server) Start(logger *zap.Logger) {
	logger.Info("starting server", zap.String("addr", srv.Addr))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srvErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			srvErr <- err
		}
	}()

	gracefulShutdown := func(reason string) {
		logger.Info("Shutting down server...\n", zap.String("reason", reason))

		dbErr := database.CloseAll()
		if dbErr != nil {
			logger.Error("Failed to close DB connections", zap.Error(dbErr))
		}
		logger.Info("ALL DB connections closed successfuly")

		srvCtx, cancelSrv := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelSrv()
		if err := srv.Shutdown(srvCtx); err != nil {
			logger.Fatal("Failed to gracefully shut down", zap.Error(err))
		}
		logger.Info("Server gracefully stopped")

		_ = logger.Sync()
	}

	select {
	case <-ctx.Done():
		gracefulShutdown("Interrupt received")
	case err := <-srvErr:
		logger.Error("", zap.Error(err))
		gracefulShutdown("Server error")
	}
}
