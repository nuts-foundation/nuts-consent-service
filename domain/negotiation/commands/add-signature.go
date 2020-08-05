package commands

import (
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
)

const AddSignatureCmdType = eventhorizon.CommandType("negotiation:add-signature")

func init() {
	eventhorizon.RegisterCommand(func() eventhorizon.Command {
		return &AddSignature{}
	})
}

type AddSignature struct {
	// NegotiationID
	ID          uuid.UUID
	ConsentHash string
	Signature   string
	PartyID     string
}

func (cmd AddSignature) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd AddSignature) AggregateType() eventhorizon.AggregateType {
	return domain.ConsentNegotiationAggregateType
}

func (cmd AddSignature) CommandType() eventhorizon.CommandType {
	return AddSignatureCmdType
}
