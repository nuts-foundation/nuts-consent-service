package consent

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"time"
)

type ConsentNegotiation struct {
	ID uuid.UUID
	Version int
	UpdatedAt time.Time
	Contract string
}

var _ = eh.Versionable(&ConsentNegotiation{})
var _ = eh.Entity(&ConsentNegotiation{})

func (entity ConsentNegotiation) AggregateVersion() int {
	return entity.Version
}

func (entity ConsentNegotiation) EntityID() uuid.UUID {
	return entity.ID
}

type SyncProjector struct {
}

func (p SyncProjector) Project(ctx context.Context, event eh.Event, entity eh.Entity) (eh.Entity, error) {
	model, ok := entity.(*ConsentNegotiation)
	if !ok {
		return nil, errors.New("model is of incorrect type")
	}

	switch event.EventType() {
	case events.Proposed:
		data, ok := event.Data().(events.ProposedData)
		if !ok {
			return nil, errors.New("event data of wrong type")
		}
		model.ID = event.AggregateID()
		model.Contract = fmt.Sprintf("custodian:%s,actor:%s,subject:%s",data.CustodianID, data.ActorID, data.SubjectID)
	case events.Unique:
	default:
		return model, fmt.Errorf("could not project event: %s", event.EventType())
	}
	model.Version++
	model.UpdatedAt = TimeNow()
	return model, nil
}

func (p SyncProjector) ProjectorType() projector.Type {
	return projector.Type("sync-projector")
}

