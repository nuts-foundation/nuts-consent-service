package consent

import (
	"context"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"log"
	"time"
)

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &ConsentAggregate{
			AggregateBase: events.NewAggregateBase(ConsentAggregateType, id),
		}
	})
}

const ConsentAggregateType = eh.AggregateType("consent")

type ConsentAggregateState string

const ConsentRequestPending = ConsentAggregateState("pending")
const ConsentRequestCompleted = ConsentAggregateState("completed")
const ConsentRequestErrored = ConsentAggregateState("errored")
const ConsentRequestCanceled = ConsentAggregateState("canceled")

var TimeNow = func() time.Time {
	return time.Now()
}

type ConsentAggregate struct {
	*events.AggregateBase

	CustodianID string
	SubjectID   string
	ActorID     string

	State ConsentAggregateState
}

func (c *ConsentAggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	log.Printf("[ConsentAggregate] command: %v, %+v\n", command.CommandType(), command)

	// Reject every command when the Consent is cancelled
	if c.State == ConsentRequestCanceled {
		return domain.ErrAggregateCancelled
	}

	switch cmd := command.(type) {
	case *Propose:
		c.StoreEvent(events2.Proposed, events2.ProposedData{
			ID:          cmd.ID,
			CustodianID: cmd.CustodianID,
			SubjectID:   cmd.SubjectID,
			ActorID:     cmd.ActorID,
			Start:       cmd.Start,
		}, TimeNow())
	case *Cancel:
		c.StoreEvent(events2.Canceled, nil, TimeNow())
	case *MarkAsUnique:
		c.StoreEvent(events2.Unique, nil, TimeNow())
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
	}
	return nil
}
