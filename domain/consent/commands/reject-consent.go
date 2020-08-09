package commands

import (
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
)

const RejectConsentCmdType = eventhorizon.CommandType("consent:reject")

func init() {
	eventhorizon.RegisterCommand(func() eventhorizon.Command {
		return &RejectConsentRequest{}
	})
}

type RejectConsentRequest struct {
	ID uuid.UUID
	Reason string
}

func (cmd RejectConsentRequest) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd RejectConsentRequest) AggregateType() eventhorizon.AggregateType {
	return domain.ConsentAggregateType
}

func (cmd RejectConsentRequest) CommandType() eventhorizon.CommandType {
	return RejectConsentCmdType
}

