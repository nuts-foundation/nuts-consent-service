package commands

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
)

const UpdateStateCmdType = eh.CommandType("negotiation:update-state")

type UpdateState struct {
	ID uuid.UUID
	State interface{}
}

func (cmd UpdateState) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd UpdateState) AggregateType() eh.AggregateType {
	return domain.ConsentNegotiationAggregateType
}

func (cmd UpdateState) CommandType() eh.CommandType {
	return UpdateStateCmdType
}

