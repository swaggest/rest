package graceful

import (
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

// WaitToShutdownFastHTTP synchronously waits for shutdown signal and shutdowns fasthttp server.
func (s *Shutdown) WaitToShutdownFastHTTP(server *fasthttp.Server, subscriber string) error {
	shutdown, done := s.ShutdownSignal(subscriber)

	<-shutdown

	s.mu.Lock()
	timeout := s.Timeout
	s.mu.Unlock()

	if timeout == 0 {
		timeout = DefaultTimeout
	}

	fs := make(chan error)

	go func() {
		fs <- server.Shutdown()
	}()

	select {
	case <-time.After(timeout):
		close(done)
		return fmt.Errorf("failed to gracefully shutdown fasthttp server in %s", timeout.String())
	case err := <-fs:
		close(done)
		return err
	}
}
