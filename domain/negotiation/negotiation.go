package negotiation

import (
	"context"
	"fmt"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	consent_utils "github.com/nuts-foundation/nuts-consent-service/consent-utils"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	nutsCryto "github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	"log"
	"time"
)

type NegotiationAggregate struct {
	*events.AggregateBase
	FactBuilder consent_utils.ConsentFactBuilder
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

func (na NegotiationAggregate) HandleCommand(ctx context.Context, command eh.Command) error {
	fmt.Printf("[NegotiationAggregate] command: %+v\n", command)
	switch cmd := command.(type) {
	case *commands.PrepareNegotiation:
		var consentFact []byte
		var err error

		data := cmd.ConsentData

		// Construct the fact
		if consentFact, err = na.FactBuilder.BuildFact(data); err != nil {
			na.StoreEvent(events2.ConsentRequestFailed, events2.FailedData{
				Reason: fmt.Sprintf("Could not build the ConsentFact: %w", err),
			}, time.Now())
		}
		log.Printf("[NegotiationAggregate] ConsentFact created: %s\n", consentFact)

		// Validate the resulting fact
		if validationResult, err := na.FactBuilder.VerifyFact(consentFact); !validationResult || err != nil {
			na.StoreEvent(events2.ConsentRequestFailed, events2.FailedData{
				Reason: fmt.Sprintf("Could not validate the ConsentFact: %w", err),
			}, time.Now())
		}
		log.Printf("[NegotiationAggregate] ConsentFact is valid")

		// Create the externalID for the combination subject, custodian and actor.
		cryptoClient := nutsCryto.NewCryptoClient()
		legalEntity := types.LegalEntity{URI: data.CustodianID}
		entityKey := types.KeyForEntity(legalEntity)
		externalID, err := cryptoClient.CalculateExternalId(data.SubjectID, data.ActorID, entityKey)

		na.StoreEvent(events2.NegotiationPrepared, events2.NegotiationData{
			ConsentID:   externalID,
			ConsentFact: consentFact,
		}, time.Now())

	}
	return nil
}

func (n NegotiationAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	fmt.Printf("[NegotiationAggregate] event: %+v\n", event)
	return nil
}
