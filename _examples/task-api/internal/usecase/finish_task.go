package usecase

import (
	"context"

	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

type finishTaskDeps interface {
	TaskFinisher() task.Finisher
}

// FinishTask creates usecase interactor.
func FinishTask(deps finishTaskDeps) usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.SetName("closeTask")
	u.SetTitle("Finish Task")
	u.SetDescription("Finish task by ID.")
	u.Input = new(task.Identity)
	u.SetExpectedErrors(
		status.NotFound,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	u.Interactor = usecase.Interact(func(ctx context.Context, input, _ interface{}) error {
		var (
			in  = input.(*task.Identity)
			err error
		)

		err = deps.TaskFinisher().Finish(ctx, *in)

		return err
	})

	return u
}
