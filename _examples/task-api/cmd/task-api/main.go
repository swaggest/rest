package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest/_examples/task-api/internal/infra"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/nethttp"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/service"
	"github.com/valyala/fasthttp"
)

func main() {
	// Initialize config from ENV vars.
	cfg := service.Config{}

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	// Initialize application resources.
	l := infra.NewServiceLocator(cfg)

	// Terminate service locator on CTRL+C (SIGTERM or SIGINT).
	l.EnableGracefulShutdown()

	// Initialize HTTP server.
	srv := fasthttp.Server{
		ReadTimeout: 9 * time.Second,
		IdleTimeout: 9 * time.Second,
		Handler:     fchi.RequestHandler(nethttp.NewRouter(l)),
	}

	// Start HTTP server.
	log.Printf("starting HTTP server at http://localhost:%d/docs\n", cfg.HTTPPort)

	go func() {
		err := srv.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort))
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for termination signal and HTTP shutdown finished.
	err := l.WaitToShutdownFastHTTP(&srv, "http")
	if err != nil {
		log.Fatal(err)
	}

	// Wait for service locator termination finished.
	err = l.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
