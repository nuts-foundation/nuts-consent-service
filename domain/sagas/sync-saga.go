package sagas

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/negotiator/local"
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

		// make sure we get the latest version
		versionedCtx, _ := eh.NewContextWithMinVersionWait(ctx, event.Version())
		entity, err := s.NegotiationRepo.Find(versionedCtx, event.AggregateID())
		if err != nil {
			panic(err)
		}
		negotiation, ok := entity.(*consent.ConsentNegotiation)
		if !ok {
			log.Panic("entity is not of type ConsentNegotiation")
		}

		log.Printf("[SyncSaga] negotiation: %+v\n", negotiation)

		syncId, err := local.LocalNegotiator{}.Start(negotiation.PartyIDs, negotiation.Contract)
		if err != nil {
			log.Printf("[SyncSaga] could not start the sync: %+v", err)
			// Todo: return command mark-as-errored
		}
		return []eh.Command{&commands.StartSync{
			ID:     event.AggregateID(),
			SyncID: syncId,
		}}
	default:
		log.Printf("[SyncSaga] unknown eventtype '%s'\n", event.EventType())
	}
	return nil
}
