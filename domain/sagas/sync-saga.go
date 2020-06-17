package sagas

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"log"
)

const SyncSagaType saga.Type = "SyncSagaType"

type SyncSaga struct {
	NegotiationRepo eh.ReadWriteRepo
}

func (s SyncSaga) SagaType() saga.Type {
	return SyncSagaType
}

func (s SyncSaga) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	log.Printf("[SyncSaga] event: %+v\n", event)

	switch event.EventType() {
	case events.Unique:
		log.Println("[SyncSaga] Consent is unique, let's sync!")

		negotiation, err := s.NegotiationRepo.Find(ctx, event.AggregateID())
		if err != nil {
			panic(err)
		}

		log.Printf("%+v\n", negotiation)

	default:
		log.Printf("[SyncSaga] unknown eventtype '%s'\n", event.EventType())
	}
	return nil
}
