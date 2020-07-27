package negotiation

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	consentutils "github.com/nuts-foundation/nuts-consent-service/consent-utils"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	domainEvents "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	nutsCryto "github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	eventOctopus "github.com/nuts-foundation/nuts-event-octopus/pkg"
	"time"
)

type NegotiationAggregate struct {
	*events.AggregateBase
	FactBuilder consentutils.ConsentFactBuilder
	EventPublisher   eventOctopus.IEventPublisher

	ConsentFact []byte
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
	logger.Logger().Tracef("[NegotiationAggregate] command: %+v\n", command)
	switch cmd := command.(type) {
	case *commands.PrepareNegotiation:
		var consentFact []byte
		var err error

		data := cmd.ConsentData

		// Construct the fact
		if consentFact, err = na.FactBuilder.BuildFact(data); err != nil {
			na.StoreEvent(domainEvents.ConsentRequestFailed, domainEvents.FailedData{
				Reason: fmt.Sprintf("Could not build the ConsentFact: %w", err),
			}, time.Now())
		}
		logger.Logger().Debugf("[NegotiationAggregate] ConsentFact created: %s\n", consentFact)

		// Validate the resulting fact
		if validationResult, err := na.FactBuilder.VerifyFact(consentFact); !validationResult || err != nil {
			na.StoreEvent(domainEvents.ConsentRequestFailed, domainEvents.FailedData{
				Reason: fmt.Sprintf("Could not validate the ConsentFact: %w", err),
			}, time.Now())
		}
		logger.Logger().Tracef("[NegotiationAggregate] ConsentFact is valid")

		na.StoreEvent(domainEvents.NegotiationPrepared, domainEvents.NegotiationData{
			ConsentFact: consentFact,
		}, time.Now())
	case *commands.ProposeConsent:
		logger.Logger().Tracef("[NegotiationAggregate]: Propose consent for ID: %s", na.negotiationID())

		channel := consentutils.CordaChannel{}
		consentFact, nil := na.FactBuilder.FactFromBytes(na.ConsentFact)

		// Create the externalID for the combination subject, custodian and actor.
		cryptoClient := nutsCryto.NewCryptoClient()
		legalEntity := types.LegalEntity{URI: consentFact.Custodian()}
		entityKey := types.KeyForEntity(legalEntity)
		binExternalID, err := cryptoClient.CalculateExternalId(consentFact.Subject(), consentFact.Actor(), entityKey)
		externalID := hex.EncodeToString(binExternalID)

		state, err := channel.BuildFullConsentRequestState(na.negotiationID(), externalID, consentFact)
		if err != nil {
			return fmt.Errorf("could not sync consent proposal: %w", err)
		}

		sjs, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("failed to marshall NewConsentRequest to json: %v", err)
		}
		bsjs := base64.StdEncoding.EncodeToString(sjs)

		cordaBridgeEvent := eventOctopus.Event{
			// Use the ID of this negotiation for the event UUID
			UUID:                 na.negotiationID().String(),
			Name:                 eventOctopus.EventConsentRequestConstructed,
			InitiatorLegalEntity: consentFact.Custodian(),
			RetryCount:           0,
			ExternalID:           externalID,
			Payload:              bsjs,
		}

		err = na.EventPublisher.Publish(eventOctopus.ChannelConsentRequest, cordaBridgeEvent)
		if err == nil {
			na.StoreEvent(domainEvents.ConsentProposed, nil, time.Now())
		}
		return err
	}
	return nil
}

func (na *NegotiationAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	logger.Logger().Tracef("[NegotiationAggregate] event: %+v\n", event)
	switch event.EventType() {
	case domainEvents.NegotiationPrepared:
		if data, ok := event.Data().(domainEvents.NegotiationData); ok {
			na.ConsentFact = data.ConsentFact
		} else {
			return fmt.Errorf("could not apply event: %w", domain.ErrInvalidEventData)
		}
	}
	return nil
}

func (na NegotiationAggregate) negotiationID() uuid.UUID {
	return na.EntityID()
}