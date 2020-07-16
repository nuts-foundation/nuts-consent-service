package commands

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
)

const MarkCustodianCheckedCmdType = eh.CommandType("consent:mark-custodian-checked")

type MarkCustodianChecked struct {
	ID uuid.UUID
}

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &MarkCustodianChecked{}
	})
}

func (cmd MarkCustodianChecked) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd MarkCustodianChecked) AggregateType() eh.AggregateType {
	return domain.ConsentAggregateType
}

func (cmd MarkCustodianChecked) CommandType() eh.CommandType {
	return MarkCustodianCheckedCmdType
}

