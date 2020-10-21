// Package task describes task domain.
package task

import "context"

// Creator creates tasks.
type Creator interface {
	Create(context.Context, Value) (Entity, error)
}

// Updater updates tasks.
type Updater interface {
	Update(context.Context, Identity, Value) error
}

// Finisher closes tasks.
type Finisher interface {
	Cancel(context.Context, Identity) error
	Finish(context.Context, Identity) error
}

// Finder finds tasks.
type Finder interface {
	Find(context.Context) []Entity
	FindByID(context.Context, Identity) (Entity, error)
}
