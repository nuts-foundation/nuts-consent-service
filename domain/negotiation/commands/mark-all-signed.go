package commands

import (
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
)

const MarkAllSignedCmdType = eventhorizon.CommandType("negotiation:mark-all-signed")

type MarkAllSigned struct {
	ID uuid.UUID
}

func init() {
	eventhorizon.RegisterCommand(func() eventhorizon.Command {
		return &MarkAllSigned{}
	})
}

func (cmd MarkAllSigned) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd MarkAllSigned) AggregateType() eventhorizon.AggregateType {
	return domain.ConsentAggregateType
}

func (cmd MarkAllSigned) CommandType() eventhorizon.CommandType {
	return MarkAllSignedCmdType
}

