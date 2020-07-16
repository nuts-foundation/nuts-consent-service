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

// ConsentAggregate represents the consistency boundary for a specific consent
type ConsentAggregate struct {
	*events.AggregateBase

	//State ConsentAggregateState
}

func (c *ConsentAggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	log.Printf("[ConsentAggregate] command: %v, %+v\n", command.CommandType(), command)

	switch cmd := command.(type) {
	case *commands.RegisterConsent:
		c.StoreEvent(events2.ConsentRequestRegistered, events2.RequestData{
			ID:          cmd.ID,
			CustodianID: cmd.CustodianID,
			SubjectID:   cmd.SubjectID,
			ActorID:     cmd.ActorID,
			Start:       cmd.Start,
		}, TimeNow())
	default:
		return domain.ErrUnknownCommand
	}
	return nil
}

func (c *ConsentAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	log.Printf("[ConsentAggregate] event: %+v\n", event)
	switch event.EventType() {
	default:

	}
	return nil
}
