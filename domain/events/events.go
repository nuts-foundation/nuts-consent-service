package events

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"time"
)

const Proposed = eh.EventType("consent:proposed")
const Canceled = eh.EventType("consent:canceled")
const Errored = eh.EventType("consent:errored")
const Unique = eh.EventType("consent:unique")
const SyncStarted = eh.EventType("consent:sync-started")
const CustodianChecked = eh.EventType("consent:custodian-checked")

type ProposedData struct {
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
	eh.RegisterEventData(Proposed, func() eh.EventData {
		return &ProposedData{}
	})

	eh.RegisterEventData(SyncStarted, func() eh.EventData {
		return &SyncStartedData{}
	})
}


