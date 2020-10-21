package graceful

import (
	"context"
	"net/http"
)

// WaitToShutdownHTTP synchronously waits for shutdown signal and shutdowns http server.
func (s *Shutdown) WaitToShutdownHTTP(server *http.Server, subscriber string) error {
	shutdown, done := s.ShutdownSignal(subscriber)

	<-shutdown

	s.mu.Lock()
	timeout := s.Timeout
	s.mu.Unlock()

	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := server.Shutdown(ctx)

	close(done)

	return err
}
