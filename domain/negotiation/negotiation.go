package negotiation

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
)

const ConsentNegotiationAggregateType = eh.AggregateType("consent-negotiation")

func init() {
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &NegotiationAggregate{
			AggregateBase: events.NewAggregateBase(ConsentNegotiationAggregateType, id),
		}
	})
}

type NegotiationAggregate struct {
	*events.AggregateBase
	Contents string

	State string
}

type PartyRole string

const CustodianRole = PartyRole("custodian")
const ActorRole = PartyRole("actor")
const SubjectRole = PartyRole("subject")

// Party keeps track of vendor responses representing this party
type Party struct {
	ID              string
	Role            PartyRole
	Vendor          []string // list of all vendors representing this party
	VendorResponses []VendorResponse
}

type VendorResponse struct {
	Signed bool
}

func (n NegotiationAggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	fmt.Printf("[NegotiationAggregate] command: %+v\n", command)
	return nil
}

func (n NegotiationAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	fmt.Printf("[NegotiationAggregate] event: %+v\n", event)
	return nil
}
