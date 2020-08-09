package commands

import (
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
)

const AddConsentCmdType = eventhorizon.CommandType("negotiation:add-consent")

func init() {
	eventhorizon.RegisterCommand(func() eventhorizon.Command {
		return &AddConsent{}
	})
}

type AddConsent struct {
	ID uuid.UUID
	ConsentData events.ConsentData
}

func (cmd AddConsent) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd AddConsent) AggregateType() eventhorizon.AggregateType {
	return domain.ConsentNegotiationAggregateType
}

func (cmd AddConsent) CommandType() eventhorizon.CommandType {
	return AddConsentCmdType
}


