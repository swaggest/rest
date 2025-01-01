//go:build go1.18

package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	var srv http.Server

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		log.Println("Shutting down...")

		// We received an interrupt signal, shut down with up to 10 seconds of waiting for current requests to finish.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		log.Println("Shutdown complete")

		close(idleConnsClosed)
	}()

	r := NewRouter()

	// You can access OpenAPI schema of an instrumented *web.Service if you need.
	j, _ := json.Marshal(r.OpenAPISchema())
	println("OpenAPI schema head:", string(j)[0:300], "...")

	srv.Handler = r
	srv.Addr = "localhost:8012"

	log.Println("http://localhost:8012/docs")
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
