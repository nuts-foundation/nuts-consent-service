package commands

import (
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
)

const PrepareNegotiationCmdType = eventhorizon.CommandType("negotiation:prepare")

func init() {
	eventhorizon.RegisterCommand(func() eventhorizon.Command {
		return PrepareNegotiation{}
	})
}

type PrepareNegotiation struct {
	ID uuid.UUID
	ConsentData events.ConsentData
}

func (cmd PrepareNegotiation) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd PrepareNegotiation) AggregateType() eventhorizon.AggregateType {
	return domain.ConsentNegotiationAggregateType
}

func (cmd PrepareNegotiation) CommandType() eventhorizon.CommandType {
	return PrepareNegotiationCmdType
}
