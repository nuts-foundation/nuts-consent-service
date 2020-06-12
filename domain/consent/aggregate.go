package consent

import (
	"context"
	"fmt"
	"github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"time"
)

const ConsentAggregateType = eventhorizon.AggregateType("consent")

type ConsentAggregateState string

const ConsentRequestPending = ConsentAggregateState("pending")
const ConsentRequestCompleted = ConsentAggregateState("completed")
const ConsentRequestErrored = ConsentAggregateState("errored")

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

func (c *ConsentAggregate) HandleCommand(ctx context.Context, command eventhorizon.Command) error {
	fmt.Printf("ConsentAggregate command: %+v", command)
	switch cmd := command.(type) {
	case *Propose:
		fmt.Printf("cmd with ID: %s\n", cmd.ID)
		c.StoreEvent(Proposed, ProposedData{
			ID:          cmd.ID,
			CustodianID: cmd.CustodianID,
			SubjectID:   cmd.SubjectID,
			ActorID:     cmd.ActorID,
			Start:       cmd.Start,
		}, TimeNow())
	case *Cancel:
		c.StoreEvent(Canceled, nil, TimeNow())
	default:
		return domain.ErrUnknownCommand
	}
	return nil
}

func (c *ConsentAggregate) ApplyEvent(ctx context.Context, event eventhorizon.Event) error {
	fmt.Printf("ConsentAggregate event: %+v\n", event)
	return nil
}
