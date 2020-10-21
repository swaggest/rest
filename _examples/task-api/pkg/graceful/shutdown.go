package graceful

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// DefaultTimeout is a default Timeout to wait for graceful termination.
const DefaultTimeout = 10 * time.Second

// Shutdown manages graceful shutdown.
type Shutdown struct {
	Timeout time.Duration

	mu             sync.Mutex
	subscribers    map[string]chan struct{}
	shutdownSignal chan struct{}
}

// Close invokes shutdown.
func (s *Shutdown) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shutdownSignal != nil {
		close(s.shutdownSignal)
		s.shutdownSignal = nil
	}
}

// Wait blocks until shutdown.
func (s *Shutdown) Wait() error {
	if s.shutdownSignal != nil {
		<-s.shutdownSignal
	}

	return s.shutdown()
}

// EnableGracefulShutdown schedules service locator termination SIGTERM or SIGINT.
func (s *Shutdown) EnableGracefulShutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shutdownSignal == nil {
		shutdownSignal := make(chan struct{})
		s.shutdownSignal = shutdownSignal

		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGTERM, syscall.SIGINT)

		go func() {
			<-exit
			close(shutdownSignal)
			s.mu.Lock()
			defer s.mu.Unlock()

			if s.shutdownSignal != nil {
				s.shutdownSignal = nil
			}
		}()
	}
}

func (s *Shutdown) shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timeout := s.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	deadline := time.After(timeout)

	for subscriber, done := range s.subscribers {
		select {
		case <-done:
			continue
		case <-deadline:
			return fmt.Errorf("shutdown deadline exceeded while waiting for %s", subscriber)
		}
	}

	return nil
}

// ShutdownSignal returns a channel that is closed when service locator is closed or os shutdownSignal is received and
// a confirmation channel that should be closed once subscriber has finished the shutdown.
func (s *Shutdown) ShutdownSignal(subscriber string) (shutdown <-chan struct{}, done chan<- struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subscribers == nil {
		s.subscribers = make(map[string]chan struct{})
	}

	if d, ok := s.subscribers[subscriber]; ok {
		return s.shutdownSignal, d
	}

	d := make(chan struct{}, 1)
	s.subscribers[subscriber] = d

	return s.shutdownSignal, d
}
