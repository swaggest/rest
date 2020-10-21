package task

import (
	"time"

	"github.com/swaggest/jsonschema-go"
)

// Status describes task state.
type Status string

// Available task statuses.
const (
	Active   = Status("")
	Canceled = Status("canceled")
	Done     = Status("done")
	Expired  = Status("expired")
)

var _ jsonschema.Exposer = Status("")

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

// Identity identifies task.
type Identity struct {
	ID int `json:"id"`
}

// Value is a task value.
type Value struct {
	Goal     string     `json:"goal" minLength:"1" required:"true"`
	Deadline *time.Time `json:"deadline,omitempty"`
}

// Entity is an identified task entity.
type Entity struct {
	Identity
	Value
	CreatedAt time.Time  `json:"createdAt"`
	Status    Status     `json:"status,omitempty"`
	ClosedAt  *time.Time `json:"closedAt,omitempty"`
}
