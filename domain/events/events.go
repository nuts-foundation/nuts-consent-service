package events

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"time"
)

const ConsentRequestRegistered = eh.EventType("consent:request-registered")
const ConsentRequestFailed = eh.EventType("consent:request-failed")
const ReservationAccepted = eh.EventType("treatment-relation:consent-reservation-accepted")
const ReservationRejected = eh.EventType("treatment-relation:consent-reservation-rejected")
const NegotiationPrepared = eh.EventType("negotiation:prepared")

type ConsentData struct {
	ID          uuid.UUID
	CustodianID string
	SubjectID   string
	ActorID     string
	Class       string
	Start       time.Time
	End         time.Time
}

type NegotiationData struct {
	ConsentID []byte
	ConsentFact []byte
}

type SyncStartedData struct {
	SyncID uuid.UUID
}

type FailedData struct {
	Reason string
}

func init() {
	eh.RegisterEventData(ConsentRequestRegistered, func() eh.EventData {
		return &ConsentData{}
	})
	eh.RegisterEventData(ReservationAccepted, func() eh.EventData {
		return &ConsentData{}
	})
	eh.RegisterEventData(NegotiationPrepared, func() eh.EventData {
		return &NegotiationData{}
	} )
}
