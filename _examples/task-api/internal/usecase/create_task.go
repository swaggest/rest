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
}) usecase.IOInteractor {
	u := usecase.NewIOI(new(task.Value), new(task.Entity), func(ctx context.Context, input, output interface{}) error {
		var (
			in  = input.(*task.Value)
			out = output.(*task.Entity)
			err error
		)

		log.Printf("creating task: %v\n", *in)
		*out, err = deps.TaskCreator().Create(ctx, *in)

		return err
	})

	u.SetDescription("Create task to be done.")
	u.SetExpectedErrors(
		status.AlreadyExists,
		status.InvalidArgument,
	)
	u.SetTags("Tasks")

	return u
}
