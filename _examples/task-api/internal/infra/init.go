package infra

import (
	"context"
	"log"
	"time"

	"github.com/swaggest/rest/_examples/task-api/internal/infra/repository"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/service"
)

// NewServiceLocator initializes application resources.
func NewServiceLocator(cfg service.Config) *service.Locator {
	l := service.Locator{}

	taskRepository := repository.Task{}

	l.TaskFinisherProvider = &taskRepository
	l.TaskFinderProvider = &taskRepository
	l.TaskUpdaterProvider = &taskRepository
	l.TaskCreatorProvider = &taskRepository

	go func() {
		if cfg.TaskCleanupInterval == 0 {
			return
		}

		shutdown, done := l.ShutdownSignal("finishExpiredTasks")

		for {
			select {
			case <-time.After(cfg.TaskCleanupInterval):
				err := taskRepository.FinishExpired(context.Background())
				if err != nil {
					log.Printf("failed to finish expired task: %v", err)
				}
			case <-shutdown:
				close(done)

				return
			}
		}
	}()

	return &l
}
