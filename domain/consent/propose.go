package consent

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"time"
)

const ProposeCmdType = eh.CommandType("consent:propose")

type Propose struct {
	ID          uuid.UUID
	CustodianID string
	SubjectID   string
	ActorID     string
	//InitiatorID string    // party(care provider or subject) who started this consent request
	//InitiatedAt time.Time // time this consent request was initiated at the initiator
	//Class       string
	//Proof       string
	Start       time.Time
	End         time.Time `eh:"optional"`
}

func init() {
	eh.RegisterCommand(func() eh.Command {
		return &Propose{}
	})
}

func (cmd Propose) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd Propose) AggregateType() eh.AggregateType {
	return ConsentAggregateType
}

func (cmd Propose) CommandType() eh.CommandType {
	return ProposeCmdType
}
