package process_managers

import (
	"context"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	"log"
)

const ConsentProgressManagerType = saga.Type("consentProgressManager")

// ConsentProgressManager manges the process of registering and synchronising a consent
// It sits between the consent aggregate and the treatment relation aggregate.
// This process manager decouples the two aggregates.
type ConsentProgressManager struct {
}

func (c ConsentProgressManager) SagaType() saga.Type {
	return ConsentProgressManagerType
}

func (c ConsentProgressManager) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	log.Printf("[ConsentProsessManager] event: %+v\n", event)
	switch event.EventType() {
	case events.ConsentRequestRegistered:
		data, ok := event.Data().(events.ConsentData)
		if !ok {
			return nil
		}
		return []eh.Command{
			&commands.ReserveConsent{
				ID:          uuid.New(),
				CustodianID: data.CustodianID,
				SubjectID:   data.SubjectID,
				ActorID:     data.ActorID,
				Class:       data.Class,
				Start:       data.Start,
				End:         data.End,
			},
		}
	}

	return nil
}
