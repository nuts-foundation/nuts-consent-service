package events

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"time"
)

const ConsentRequestRegistered = eh.EventType("consent:request-registered")

type RequestData struct {
	ID          uuid.UUID
	CustodianID string
	SubjectID   string
	ActorID     string
	Start       time.Time
}

type SyncStartedData struct {
	SyncID uuid.UUID
}

func init() {
	eh.RegisterEventData(ConsentRequestRegistered, func() eh.EventData {
		return &RequestData{}
	})
}


