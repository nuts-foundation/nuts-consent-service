package consent

import (
	"context"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"log"
	"time"
)

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &ConsentAggregate{
			AggregateBase: events.NewAggregateBase(domain.ConsentAggregateType, id),
		}
	})
}


type ConsentAggregateState string

const ConsentRequestPending = ConsentAggregateState("pending")
const ConsentRequestCompleted = ConsentAggregateState("completed")
const ConsentRequestErrored = ConsentAggregateState("errored")
const ConsentRequestCanceled = ConsentAggregateState("canceled")

var TimeNow = func() time.Time {
	return time.Now()
}

// ConsentAggregate represents the consistency boundary for a set of consents
// for a specific set of parties: a DataCustodian, Patient and a selected actor
// who can access the data.
// What makes one consent unique from another, apart from the parties, is the
// scope and the Period. There can not be any overlap in the period for a consent
// with a specific scope.
type ConsentAggregate struct {
	*events.AggregateBase

	State ConsentAggregateState
}

func (c *ConsentAggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	log.Printf("[ConsentAggregate] command: %v, %+v\n", command.CommandType(), command)

	// Reject every command when the Consent is cancelled
	if c.State == ConsentRequestCanceled {
		return domain.ErrAggregateCancelled
	}

	switch cmd := command.(type) {
	case *commands.MarkAsErrored:
		log.Printf("consent marked as errord with reason %s\n", cmd.Reason)
		c.StoreEvent(events2.Errored, nil, TimeNow())
	case *commands.Propose:
		c.StoreEvent(events2.Proposed, events2.ProposedData{
			ID:          cmd.ID,
			CustodianID: cmd.CustodianID,
			SubjectID:   cmd.SubjectID,
			ActorID:     cmd.ActorID,
			Start:       cmd.Start,
		}, TimeNow())
	case *commands.Cancel:
		c.StoreEvent(events2.Canceled, nil, TimeNow())
	case *commands.MarkAsUnique:
		c.StoreEvent(events2.Unique, nil, TimeNow())
	case *commands.StartSync:
		c.StoreEvent(events2.SyncStarted, events2.SyncStartedData{SyncID: cmd.SyncID}, TimeNow())
	case *commands.MarkCustodianChecked:
		c.StoreEvent(events2.CustodianChecked, nil, TimeNow())
	default:
		return domain.ErrUnknownCommand
	}
	return nil
}

func (c *ConsentAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	log.Printf("[ConsentAggregate] event: %+v\n", event)
	switch event.EventType() {
	case events2.Canceled:
		c.State = ConsentRequestCanceled
	case events2.Errored:
		c.State = ConsentRequestErrored
	}
	return nil
}
