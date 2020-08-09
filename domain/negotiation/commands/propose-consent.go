package commands

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
)

const ProposeConsentFactCmdType = eh.CommandType("negotiation:propose-fact")

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &ProposeConsent{}
	})
}

type ProposeConsent struct {
	ID uuid.UUID
}

func (cmd ProposeConsent) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd ProposeConsent) AggregateType() eh.AggregateType {
	return domain.ConsentNegotiationAggregateType
}

func (cmd ProposeConsent) CommandType() eh.CommandType {
	return ProposeConsentFactCmdType
}

