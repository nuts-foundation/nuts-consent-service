package consent

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
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
	return ConsentAggregateType
}

func (cmd MarkCustodianChecked) CommandType() eh.CommandType {
	return MarkCustodianCheckedCmdType
}

