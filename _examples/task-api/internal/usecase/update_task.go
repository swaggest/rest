package usecase

import (
	"context"

	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

type updateTask struct {
	task.Identity `json:"-"`
	task.Value
}

// UpdateTask creates usecase interactor.
func UpdateTask(deps interface {
	TaskUpdater() task.Updater
}) usecase.Interactor {
	u := usecase.IOInteractor{}

	u.SetName("updateTask")
	u.SetTitle("Update Task")
	u.SetDescription("Update existing task.")
	u.Input = new(updateTask)
	u.SetExpectedErrors(
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	u.Interactor = usecase.Interact(func(ctx context.Context, input, _ interface{}) error {
		var (
			in  = input.(*updateTask)
			err error
		)

		err = deps.TaskUpdater().Update(ctx, in.Identity, in.Value)

		return err
	})

	return u
}
