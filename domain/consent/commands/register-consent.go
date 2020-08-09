package commands

import (
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"time"
)

const RegisterConsentCmdType = eventhorizon.CommandType("consent:register")

func init() {
	eventhorizon.RegisterCommand(func() eventhorizon.Command {
		return RegisterConsent{}
	})
}

type RegisterConsent struct {
	ID          uuid.UUID
	CustodianID string
	SubjectID   string
	ActorID     string
	Class       string
	Start       time.Time
	End         time.Time `eh:"optional"`
	//InitiatorID string    // party(care provider or subject) who started this consent request
	//InitiatedAt time.Time // time this consent request was initiated at the initiator
	//Proof       string
}

func (cmd RegisterConsent) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd RegisterConsent) AggregateType() eventhorizon.AggregateType {
	return domain.ConsentAggregateType
}

func (cmd RegisterConsent) CommandType() eventhorizon.CommandType {
	return RegisterConsentCmdType
}

