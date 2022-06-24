package usecase

import (
	"context"
	"fmt"

	"github.com/swaggest/rest-fasthttp/_examples/task-api/internal/domain/task"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// FindTasks creates usecase interactor.
func FindTasks(
	deps interface {
		TaskFinder() task.Finder
	},
) usecase.IOInteractor {
	u := usecase.NewIOI(nil, new([]task.Entity), func(ctx context.Context, input, output interface{}) error {
		out, ok := output.(*[]task.Entity)
		if !ok {
			return fmt.Errorf("%w: unexpected output type %T", status.Unimplemented, output)
		}

		*out = deps.TaskFinder().Find(ctx)

		return nil
	})

	u.SetDescription("Find all tasks.")
	u.Output = new([]task.Entity)
	u.SetTags("Tasks")

	return u
}
