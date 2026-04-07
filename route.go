package rest

import (
	"github.com/swaggest/usecase"
)

// HandlerWithRoute is a http.Handler with routing information.
type HandlerWithRoute interface {
	// RouteMethod returns http method of action.
	RouteMethod() string

	// RoutePattern returns http path pattern of action.
	RoutePattern() string
}

// HandlerWithUseCase exposes usecase.
type HandlerWithUseCase interface {
	UseCase() usecase.Interactor
}
