package task

import (
	"time"

	"github.com/swaggest/jsonschema-go"
)

// Available task statuses.
const (
	Active   = Status("")
	Canceled = Status("canceled")
	Done     = Status("done")
	Expired  = Status("expired")
)

// Entity is an identified task entity.
type Entity struct {
	Identity
	Value
	CreatedAt time.Time  `json:"createdAt"`
	Status    Status     `json:"status,omitempty"`
	ClosedAt  *time.Time `json:"closedAt,omitempty"`
}

// Identity identifies task.
type Identity struct {
	ID int `json:"id"`
}

// Status describes task state.
type Status string

// JSONSchema exposes Status JSON schema, implements jsonschema.Exposer.
func (Status) JSONSchema() (jsonschema.Schema, error) {
	s := jsonschema.Schema{}
	s.
		WithType(jsonschema.String.Type()).
		WithTitle("Goal Status").
		WithDescription("Non-empty task status indicates result.").
		WithEnum(Active, Canceled, Done, Expired)

	return s, nil
}

// Value is a task value.
type Value struct {
	Goal     string     `json:"goal" minLength:"1" required:"true"`
	Deadline *time.Time `json:"deadline,omitempty"`
}

var _ jsonschema.Exposer = Status("")
