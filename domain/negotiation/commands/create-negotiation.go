package commands

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
)

const CreateNegotiationCmdType = eh.CommandType("negotiation:create")

type CreateNegotiation struct {
	ID                    uuid.UUID
	ExternalNegotiationID string
	CustodianID           string
	ActorID               string
	SubjectID             string
}

func (cmd CreateNegotiation) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd CreateNegotiation) AggregateType() eh.AggregateType {
	return domain.ConsentNegotiationAggregateType
}

func (cmd CreateNegotiation) CommandType() eh.CommandType {
	return CreateNegotiationCmdType
}
