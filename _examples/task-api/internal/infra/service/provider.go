package service

import "github.com/swaggest/rest-fasthttp/_examples/task-api/internal/domain/task"

// TaskCreatorProvider is a service locator provider.
type TaskCreatorProvider interface {
	TaskCreator() task.Creator
}

// TaskUpdaterProvider is a service locator provider.
type TaskUpdaterProvider interface {
	TaskUpdater() task.Updater
}

// TaskFinderProvider is a service locator provider.
type TaskFinderProvider interface {
	TaskFinder() task.Finder
}

// TaskFinisherProvider is a service locator provider.
type TaskFinisherProvider interface {
	TaskFinisher() task.Finisher
}
