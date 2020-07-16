package commands

import (
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"time"
)

const ReserveConsentCmdType = eventhorizon.CommandType("treatment-relation:reserve-consent")

func init() {
	eventhorizon.RegisterCommand(func() eventhorizon.Command {
		return ReserveConsent{}
	})
}

// ReserveConsent commands the treatmentRelationAggregate to reserve a consent for the combination
// of parties, scope and period.
// http://www.rgoarchitects.com/Files/SOAPatterns/ReservationPattern.pdf
type ReserveConsent struct {
	ID uuid.UUID

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

func (cmd ReserveConsent) AggregateID() uuid.UUID {
	return cmd.ID
}

func (cmd ReserveConsent) AggregateType() eventhorizon.AggregateType {
	return domain.TreatmentRelationAggregateType
}

func (cmd ReserveConsent) CommandType() eventhorizon.CommandType {
	return ReserveConsentCmdType
}


