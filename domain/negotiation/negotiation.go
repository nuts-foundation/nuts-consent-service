package negotiation

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	consentutils "github.com/nuts-foundation/nuts-consent-service/consent-utils"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	domainEvents "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	eventOctopus "github.com/nuts-foundation/nuts-event-octopus/pkg"
	"time"
)

var TimeNow = func() time.Time {
	return time.Now()
}

type NegotiationAggregate struct {
	// Services:
	*events.AggregateBase
	FactBuilder    consentutils.ConsentFactBuilder
	EventPublisher eventOctopus.IEventPublisher
	Channel        consentutils.SyncChannel

	// Negotiation data:
	externalNegotiationID string
	subjectID             string
	custodianID           string
	actorID               string

	Signatures   map[string]map[string]string
	ConsentFacts [][]byte
	State        interface{}
}

type PartyRole string

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
	logger.Logger().Debugf("[NegotiationAggregate] command: %+v\n", command)
	switch cmd := command.(type) {
	case *commands.CreateNegotiation:
		if na.externalNegotiationID != "" {
			return fmt.Errorf("negotiation already created")
		}
		na.StoreEvent(domainEvents.NegotiationCreated, domainEvents.NegotiationBaseData{
			ExternalNegotiationData: cmd.ExternalNegotiationID,
			CustodianID:             cmd.CustodianID,
			SubjectID:               cmd.SubjectID,
			ActorID:                 cmd.ActorID,
		}, TimeNow())
	case *commands.UpdateState:
		na.StoreEvent(domainEvents.NegotiationStateUpdated, domainEvents.ChannelStateData{State: cmd.State}, TimeNow())
	case *commands.AddConsent:
		// TODO check if the custodian, subject and actor are equal to this negotiation
		var consentFact []byte
		var err error

		// Construct the fact
		if consentFact, err = na.FactBuilder.BuildFact(cmd.ConsentData); err != nil {
			na.StoreEvent(domainEvents.ConsentRequestFailed, domainEvents.FailedData{
				Reason: fmt.Sprintf("Could not build the ConsentFact: %w", err),
			}, TimeNow())
		}
		logger.Logger().Tracef("[NegotiationAggregate] ConsentFact created: %s\n", consentFact)

		// Validate the resulting fact
		if validationResult, err := na.FactBuilder.VerifyFact(consentFact); !validationResult || err != nil {
			na.StoreEvent(domainEvents.ConsentRequestFailed, domainEvents.FailedData{
				Reason: fmt.Sprintf("Could not validate the ConsentFact: %w", err),
			}, TimeNow())
		}
		logger.Logger().Tracef("[NegotiationAggregate] ConsentFact is valid")

		na.StoreEvent(domainEvents.ConsentFactGenerated, domainEvents.ConsentFactData{
			ConsentID:   cmd.ConsentData.ID,
			ConsentFact: consentFact,
		}, TimeNow())
	case *commands.ProposeConsent:
		logger.Logger().Debugf("[NegotiationAggregate]: Propose consent for ID: %s", na.negotiationID())

		var consentFacts []consentutils.ConsentFact

		for _, factBytes := range na.ConsentFacts {
			consentFact, _ := na.FactBuilder.FactFromBytes(factBytes)
			consentFacts = append(consentFacts, consentFact)
		}

		err := na.Channel.StartSync(na.negotiationID(), na.externalNegotiationID, na.custodianID, consentFacts)
		if err != nil {
			return fmt.Errorf("could not sync consent proposal: %w", err)
		}
		na.StoreEvent(domainEvents.ConsentProposed, nil, TimeNow())

		return err
	case *commands.AddSignature:
		logger.Logger().Debugf("[NegotiationAggregate]: Add signature to negotiation with ID: %s", na.negotiationID())

		// Check if signature is not already present
		if na.Signatures[cmd.ConsentHash] == nil || na.Signatures[cmd.ConsentHash][cmd.PartyID] == "" {
			event := na.StoreEvent(domainEvents.SignatureAdded, domainEvents.SignatureData{
				SigningParty: cmd.PartyID,
				ConsentID:    cmd.ConsentHash,
				Signature:    cmd.Signature,
			}, TimeNow())
			// apply event to the aggregate
			if err := na.ApplyEvent(ctx, event); err != nil {
				return err
			}
		}

		// Now check if all signatures are present:
		present := true
		// TODO fix this naive way of checking all signatures.
		for _, sig := range na.Signatures {
			if len(sig) != 2 {
				present = false
			}
		}

		if present { // TODO Fix check if this node is the initiating one
			logger.Logger().Debugf("[NegotiationAggregate]: all present: %s", na.negotiationID())
			cordaEvent, ok := na.State.(eventOctopus.Event)
			if !ok {
				return fmt.Errorf("could cast corda event state: %w", domain.ErrInvalidEventData)
			}
			cordaEvent.Name = eventOctopus.EventAllSignaturesPresent
			if err := na.Channel.Publish(eventOctopus.ChannelConsentRequest, &cordaEvent); err != nil {
				return err
			}

			na.StoreEvent(domainEvents.AllSignaturesPresent, nil, TimeNow())
		}
	//case *commands.MarkAllSigned:
	//	logger.Logger().Debugf("[NegotiationAggregate]: trying to mark as all signed: %s", na.negotiationID())
	default:
		return fmt.Errorf("could not handle command: '%s', %w", cmd.CommandType(), domain.ErrUnknownCommand)
	}
	return nil
}

func (na *NegotiationAggregate) ApplyEvent(ctx context.Context, event eh.Event) error {
	logger.Logger().Debugf("[NegotiationAggregate - %s] Hydrating aggregate with event: %+v\n", na.negotiationID(), event)
	switch event.EventType() {
	case domainEvents.NegotiationCreated:
		if data, ok := event.Data().(domainEvents.NegotiationBaseData); ok {
			na.externalNegotiationID = data.ExternalNegotiationData
			na.custodianID = data.CustodianID
			na.subjectID = data.SubjectID
			na.actorID = data.ActorID
		}
	case domainEvents.ConsentFactGenerated:
		if data, ok := event.Data().(domainEvents.ConsentFactData); ok {
			na.ConsentFacts = append(na.ConsentFacts, data.ConsentFact)
			//logger.Logger().Debugf("[NegotiationAggregate] adding consentFact %+v\n", data.ConsentID)
		} else {
			return fmt.Errorf("could not apply event: %w", domain.ErrInvalidEventData)
		}
	case domainEvents.ConsentProposed:
	case domainEvents.SignatureAdded:
		if data, ok := event.Data().(domainEvents.SignatureData); ok {
			if na.Signatures[data.ConsentID] == nil {
				na.Signatures[data.ConsentID] = map[string]string{}
			}
			na.Signatures[data.ConsentID][data.SigningParty] = data.Signature
		} else {
			return fmt.Errorf("could not apply event: %w", domain.ErrInvalidEventData)
		}
	case domainEvents.NegotiationStateUpdated:
		if data, ok := event.Data().(domainEvents.ChannelStateData); ok {
			na.State = data.State
		} else {
			return fmt.Errorf("could not apply event: %w", domain.ErrInvalidEventData)
		}
	}
	return nil
}

func (na NegotiationAggregate) negotiationID() uuid.UUID {
	return na.EntityID()
}
