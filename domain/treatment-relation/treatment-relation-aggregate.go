package treatment_relation

import (
	"context"
	"fmt"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	domainEvents "github.com/nuts-foundation/nuts-consent-service/domain/events"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	"time"
)

// TimeNow returns the current time. This can be overwritten during tests
var TimeNow = func() time.Time {
	return time.Now()
}

type TreatmentRelationAggregate struct {
	*events.AggregateBase
}

func (t TreatmentRelationAggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	logger.Logger().Debugf("[TreatmentRelationAggregate] command: %v, %+v\n", command.CommandType(), command)
	switch cmd := command.(type) {
	case *treatmentRelationCommands.ReserveConsent:
		// TODO: check if there are no duplicates with other consents
		t.StoreEvent(domainEvents.ReservationAccepted, domainEvents.ConsentData{
			ID:          cmd.ConsentID,
			CustodianID: cmd.CustodianID,
			SubjectID:   cmd.SubjectID,
			ActorID:     cmd.ActorID,
			Class:       cmd.Class,
			Start:       cmd.Start,
			End:         cmd.End,
		}, TimeNow())
	default:
		return fmt.Errorf("[TreatmentRelationAggregate] could not handle command '%s': %w\n",command.CommandType(), domain.ErrUnknownCommand)
	}
	return nil
}

func (t TreatmentRelationAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	logger.Logger().Debugf("[TreatmentRelationAggregate] event: %+v\n", event)
	switch event.EventType() {
	default:

	}
	return nil
}
