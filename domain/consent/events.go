package consent

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"time"
)

const Proposed = eh.EventType("consent:proposed")
const Canceled = eh.EventType("consent:canceled")

type ProposedData struct {
	ID          uuid.UUID
	CustodianID string
	SubjectID   string
	ActorID     string
	Start       time.Time
}

func init() {
	eh.RegisterEventData(Proposed, func() eh.EventData {
		return &ProposedData{}
	})
}


