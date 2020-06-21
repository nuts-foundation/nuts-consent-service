package consent

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

const MarkAsErroredCmdType = eh.CommandType("consent:mark-as-errored")

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &MarkAsErrored{}
	})
}

type MarkAsErrored struct {
	ID     uuid.UUID
	Reason string
}

func (cmd MarkAsErrored) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd MarkAsErrored) AggregateType() eh.AggregateType {
	return ConsentAggregateType
}

func (cmd MarkAsErrored) CommandType() eh.CommandType {
	return MarkAsErroredCmdType
}
