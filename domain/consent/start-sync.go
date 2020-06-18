package consent

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

const StartSyncCmdType = eh.CommandType("consent:start-sync")

type StartSync struct {
	ID uuid.UUID
	SyncID uuid.UUID
}

func (cmd StartSync) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd StartSync) AggregateType() eh.AggregateType {
	return ConsentAggregateType
}

func (cmd StartSync) CommandType() eh.CommandType {
	return StartSyncCmdType
}

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &StartSync{}
	})
}
