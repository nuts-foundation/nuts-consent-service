package consent

import (
	"context"
	"fmt"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	"time"
)

var TimeNow = func() time.Time {
	return time.Now()
}

// ConsentAggregate represents the consistency boundary for a specific consent
type ConsentAggregate struct {
	*events.AggregateBase
}

func (c *ConsentAggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	logger.Logger().Debugf("[ConsentAggregate] command: %v, %+v\n", command.CommandType(), command)

	switch cmd := command.(type) {
	case *commands.RegisterConsent:
		c.StoreEvent(events2.ConsentRequestRegistered, events2.ConsentData{
			ID:          cmd.ID,
			CustodianID: cmd.CustodianID,
			SubjectID:   cmd.SubjectID,
			ActorID:     cmd.ActorID,
			Class:       cmd.Class,
			Start:       cmd.Start,
			End:         cmd.End,
		}, TimeNow())
	default:
		return fmt.Errorf("[ConsentAggregate] could not handle command '%s': %w\n", command.CommandType(), domain.ErrUnknownCommand)
	}
	return nil
}

func (c *ConsentAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	logger.Logger().Debugf("[ConsentAggregate] event: %+v\n", event)
	switch event.EventType() {
	default:

	}
	return nil
}
