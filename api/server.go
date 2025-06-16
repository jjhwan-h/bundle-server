package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"bundle-server/api/app/router"
	"bundle-server/database"
)

const (
	ReadHeaderTimeout = 5
	IdleTimeout       = 30
)

type Server struct {
	*http.Server
}

func NewServer(port string) (*Server, error) {
	log.Println("configuring server...")
	api, err := router.New()
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

func (srv *Server) Start() {
	log.Println("starting server...")
	log.Println(srv.Addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srvErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			srvErr <- err
		}
	}()

	gracefulShutdown := func(reason string) {
		log.Printf("[INFO] %s / Shutting down server...\n", reason)

		dbErr := database.CloseAll()
		if dbErr != nil {
			log.Printf("%s: %v", "[ERROR] failed to close DB connections", dbErr)
		}
		log.Println("[INFO] ALL DB connections closed successfuly")

		srvCtx, cancelSrv := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelSrv()
		if err := srv.Shutdown(srvCtx); err != nil {
			panic(err)
		}
		log.Println("[INFO] Server gracefully stopped")
	}

	select {
	case <-ctx.Done():
		gracefulShutdown("Interrupt received")
	case err := <-srvErr:
		log.Printf("[ERROR] %v\n", err)
		gracefulShutdown("Server error")
	}
}
