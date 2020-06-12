package consent

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

const CancelCmdType = eh.CommandType("consent:cancel")

type Cancel struct {
	ID uuid.UUID
	Reason string
}

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &Cancel{}
	})
}

func (c Cancel) AggregateID() uuid.UUID {
	return c.ID
}

func (c Cancel) AggregateType() eh.AggregateType {
	return ConsentAggregateType
}

func (c Cancel) CommandType() eh.CommandType {
	return CancelCmdType
}

