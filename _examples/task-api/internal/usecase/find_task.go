package usecase

import (
	"context"

	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// FindTask creates usecase interactor.
func FindTask(deps interface {
	TaskFinder() task.Finder
}) usecase.Interactor {
	u := usecase.IOInteractor{}

	u.SetName("findTask")
	u.SetTitle("Find Task")
	u.SetDescription("Find task by ID.")
	u.Input = new(task.Identity)
	u.Output = new(task.Entity)
	u.SetExpectedErrors(
		status.NotFound,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		var (
			in  = input.(*task.Identity)
			out = output.(*task.Entity)
			err error
		)

		*out, err = deps.TaskFinder().FindByID(ctx, *in)

		return err
	})

	return u
}
