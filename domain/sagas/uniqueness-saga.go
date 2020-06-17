package sagas

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"log"
)

type UniquenessSaga struct {
	existingIds []string
}

func NewUniquenessSaga() *UniquenessSaga {
	return &UniquenessSaga{existingIds: make([]string, 0)}
}

const UniquenessSagaType saga.Type = "ConsentUniquenessSaga"

func (s UniquenessSaga) SagaType() saga.Type {
	return UniquenessSagaType
}

func (s *UniquenessSaga) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	log.Printf("[UniquenessSaga] event: %+v\n", event)
	switch event.EventType() {
	case events.Proposed:
		data, ok := event.Data().(events.ProposedData)
		if ok {
			id := data.CustodianID + data.SubjectID + data.ActorID
			log.Printf("[UniquenessSaga] checking duplicates for: %v\n", id)
			for _, existingId := range s.existingIds {
				log.Printf("[UniquenessSaga] against known id: %v\n", existingId)
				if id == existingId {
					log.Println("[UniquenessSaga] duplicate found!")
					return []eh.Command{&consent.Cancel{
						ID:     event.AggregateID(),
						Reason: "duplicate consent",
					}}
				}
			}
			s.existingIds = append(s.existingIds, id)
			return []eh.Command{&consent.MarkAsUnique{
				ID: event.AggregateID(),
			}}
		}

	}
	return nil
}
