package consent

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

const MarkAsUniqueCmdType = eh.CommandType("consent:mark-as-unique")

type MarkAsUnique struct {
	ID          uuid.UUID
}

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &MarkAsUnique{}
	})
}

func (cmd MarkAsUnique) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd MarkAsUnique) AggregateType() eh.AggregateType {
	return ConsentAggregateType
}

func (cmd MarkAsUnique) CommandType() eh.CommandType {
	return MarkAsUniqueCmdType
}

