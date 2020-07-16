package events

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"time"
)

const ConsentRequestRegistered = eh.EventType("consent:request-registered")
const ReservationAccepted = eh.EventType("treatment-relation:consent-reservation-accepted")
const ReservationRejected = eh.EventType("treatment-relation:consent-reservation-rejected")

type ConsentData struct {
	ID          uuid.UUID
	CustodianID string
	SubjectID   string
	ActorID     string
	Class       string
	Start       time.Time
	End         time.Time
}

type SyncStartedData struct {
	SyncID uuid.UUID
}

func init() {
	eh.RegisterEventData(ConsentRequestRegistered, func() eh.EventData {
		return &ConsentData{}
	})
	eh.RegisterEventData(ReservationAccepted, func() eh.EventData {
		return &ConsentData{}
	})
}
