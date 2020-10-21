package usecase

import (
	"context"
	"log"

	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// CreateTask creates usecase interactor.
func CreateTask(deps interface {
	TaskCreator() task.Creator
}) usecase.Interactor {
	u := struct {
		usecase.Interactor
		usecase.Info
		usecase.WithInput
		usecase.WithOutput
	}{}

	u.SetName("createTask")
	u.SetTitle("Create Task")
	u.SetDescription("Create task to be done.")
	u.Input = new(task.Value)
	u.Output = new(task.Entity)
	u.SetExpectedErrors(
		status.AlreadyExists,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	u.Interactor = usecase.Interact(func(ctx context.Context, input, output interface{}) error {
		var (
			in  = input.(*task.Value)
			out = output.(*task.Entity)
			err error
		)

		log.Printf("creating task: %v\n", *in)
		*out, err = deps.TaskCreator().Create(ctx, *in)

		return err
	})

	return u
}
