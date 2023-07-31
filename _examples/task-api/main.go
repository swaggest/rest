package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/swaggest/rest/_examples/task-api/internal/infra"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/nethttp"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/service"
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
	srv := http.Server{
		Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: nethttp.NewRouter(l),
		ReadHeaderTimeout: time.Second,
	}

	// Start HTTP server.
	log.Printf("starting HTTP server at http://localhost:%d/docs\n", cfg.HTTPPort)

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for termination signal and HTTP shutdown finished.
	err := l.WaitToShutdownHTTP(&srv, "http")
	if err != nil {
		log.Fatal(err)
	}

	// Wait for service locator termination finished.
	err = l.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
