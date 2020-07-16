package commands

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
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
	return domain.ConsentAggregateType
}

func (cmd StartSync) CommandType() eh.CommandType {
	return StartSyncCmdType
}

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &StartSync{}
	})
}
