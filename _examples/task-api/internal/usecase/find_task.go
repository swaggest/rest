package usecase

import (
	"context"

	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// FindTask creates usecase interactor.
func FindTask(
	deps interface {
		TaskFinder() task.Finder
	},
) usecase.IOInteractor {
	u := usecase.NewIOI(new(task.Identity), new(task.Entity),
		func(ctx context.Context, input, output interface{}) error {
			var (
				in  = input.(*task.Identity)
				out = output.(*task.Entity)
				err error
			)

			*out, err = deps.TaskFinder().FindByID(ctx, *in)

			return err
		})

	u.SetDescription("Find task by ID.")
	u.SetExpectedErrors(
		status.NotFound,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	return u
}
