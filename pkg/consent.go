package pkg

import (
	"context"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/eventbus/local"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/looplab/eventhorizon/eventstore/memory"
	consent_utils "github.com/nuts-foundation/nuts-consent-service/consent-utils"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent"
	consentCommands "github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	domainEvents "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation"
	negotiationCommands "github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	process_managers "github.com/nuts-foundation/nuts-consent-service/domain/process-managers"
	treatment_relation "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	nutsConsentStoreClient "github.com/nuts-foundation/nuts-consent-store/client"
	nutsConsentStorePkg "github.com/nuts-foundation/nuts-consent-store/pkg"
	nutsCryptoPkg "github.com/nuts-foundation/nuts-crypto/pkg"
	nutsEventOctClient "github.com/nuts-foundation/nuts-event-octopus/client"
	nutsEventOctopus "github.com/nuts-foundation/nuts-event-octopus/pkg"
	core "github.com/nuts-foundation/nuts-go-core"
	registryClient "github.com/nuts-foundation/nuts-registry/client"
	registry "github.com/nuts-foundation/nuts-registry/pkg"
	"log"
	"sync"
	"time"
)

type ConsentServiceConfig struct {
}

type ConsentServiceClient interface {
	StartConsentFlow(*CreateConsentRequest) (*uuid.UUID, error)
	HandleIncomingCordaEvent(event *nutsEventOctopus.Event)
}

type ConsentService struct {
	NutsRegistry     registry.RegistryClient
	NutsCrypto       nutsCryptoPkg.Client
	NutsConsentStore nutsConsentStorePkg.ConsentStoreClient
	NutsEventOctopus nutsEventOctopus.EventOctopusClient
	Config           ConsentServiceConfig
	EventPublisher   nutsEventOctopus.IEventPublisher
	CommandBus       eh.CommandHandler
}

var instance *ConsentService
var oneEngine sync.Once

func ConsentServiceInstance() *ConsentService {
	oneEngine.Do(func() {
		instance = &ConsentService{}
	})
	return instance
}

func (cl ConsentService) StartConsentFlow(request *CreateConsentRequest) (*uuid.UUID, error) {
	end := time.Time{}
	if pEnd := request.Records[0].Period.End; pEnd != nil {
		end = *pEnd
	}
	uuid := uuid.New()
	cmd := &consentCommands.RegisterConsent{
		ID:          uuid,
		CustodianID: string(request.Custodian),
		SubjectID:   string(request.Subject),
		ActorID:     string(request.Actor),
		Class:       string(request.Records[0].DataClass[0]),
		Start:       request.Records[0].Period.Start,
		End:         end,
	}
	err := cl.CommandBus.HandleCommand(context.Background(), cmd)
	return &uuid, err

}

func (cl ConsentService) HandleIncomingCordaEvent(event *nutsEventOctopus.Event) {
	logger.Logger().Debugf("incomming corda event: %+v'n", event)
}

func (cl ConsentService) Configure() error {
	return nil
}

func (cl *ConsentService) Start() error {
	cl.NutsCrypto = nutsCryptoPkg.NewCryptoClient()
	cl.NutsRegistry = registryClient.NewRegistryClient()
	cl.NutsConsentStore = nutsConsentStoreClient.NewConsentStoreClient()
	cl.NutsEventOctopus = nutsEventOctClient.NewEventOctopusClient()
	// This module has no mode feature (server/registryClient) so we delegate it completely to the global mode
	if core.NutsConfig().GetEngineMode("") != core.ServerEngineMode {
		return nil
	}
	publisher, err := cl.NutsEventOctopus.EventPublisher("consent-service")
	if err != nil {
		logger.Logger().WithError(err).Panic("Could not subscribe to event publisher")
	}
	cl.EventPublisher = publisher

	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &consent.ConsentAggregate{
			AggregateBase: events.NewAggregateBase(domain.ConsentAggregateType, id),
		}
	})
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &treatment_relation.TreatmentRelationAggregate{
			AggregateBase: events.NewAggregateBase(domain.TreatmentRelationAggregateType, id),
		}
	})

	eventstore := memory.NewEventStore()
	eventbus := local.NewEventBus(local.NewGroup())
	commandBus := bus.NewCommandHandler()
	cl.CommandBus = commandBus
	fhirBuilder := consent_utils.FhirConsentFactBuilder{}

	cordaChannel := consent_utils.CordaChannel{
		Registry:   cl.NutsRegistry,
		NutsCrypto: cl.NutsCrypto,
		Publisher:  publisher,
		CommandBus: cl.CommandBus,
		FactBuilder: fhirBuilder,
	}

	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &negotiation.NegotiationAggregate{
			AggregateBase:  events.NewAggregateBase(domain.ConsentNegotiationAggregateType, id),
			FactBuilder:    fhirBuilder,
			EventPublisher: publisher,
			Channel:        cordaChannel,
			Signatures:     map[string]map[string]string{},
		}
	})

	eventLogger := &logger.EventLogger{}
	eventbus.AddObserver(eh.MatchAny(), eventLogger)

	aggregateStore, err := events.NewAggregateStore(eventstore, eventbus)
	if err != nil {
		log.Fatal(err)
	}

	consentCommandHandler, err := aggregate.NewCommandHandler(domain.ConsentAggregateType, aggregateStore)
	treatmentCommandHander, err := aggregate.NewCommandHandler(domain.TreatmentRelationAggregateType, aggregateStore)
	negotiationCommandHandler, err := aggregate.NewCommandHandler(domain.ConsentNegotiationAggregateType, aggregateStore)

	if commandBus.SetHandler(consentCommandHandler, consentCommands.RegisterConsentCmdType) != nil ||
		commandBus.SetHandler(treatmentCommandHander, treatmentRelationCommands.ReserveConsentCmdType) != nil ||
		commandBus.SetHandler(consentCommandHandler, consentCommands.RejectConsentCmdType) != nil ||
		commandBus.SetHandler(negotiationCommandHandler, negotiationCommands.ProposeConsentFactCmdType) != nil ||
		commandBus.SetHandler(negotiationCommandHandler, negotiationCommands.AddConsentCmdType) != nil ||
		commandBus.SetHandler(negotiationCommandHandler, negotiationCommands.MarkAllSignedCmdType) != nil ||
		commandBus.SetHandler(negotiationCommandHandler, negotiationCommands.UpdateStateCmdType) != nil ||
		commandBus.SetHandler(negotiationCommandHandler, negotiationCommands.CreateNegotiationCmdType) != nil ||
		commandBus.SetHandler(negotiationCommandHandler, negotiationCommands.AddSignatureCmdType) != nil {
		panic("could not set handler")
	}

	consentProgressManager := saga.NewEventHandler(process_managers.ConsentProgressManager{}, commandBus)
	eventbus.AddHandler(eh.MatchAnyEventOf(
		domainEvents.ConsentRequestRegistered,
		domainEvents.ReservationAccepted,
		domainEvents.NegotiationPrepared,
		domainEvents.ConsentProposed,
		domainEvents.SignatureAdded,
	), consentProgressManager)

	// TODO: handle these by emitting commands
	err = cl.NutsEventOctopus.Subscribe("consent-service",
		nutsEventOctopus.ChannelConsentRequest,
		map[string]nutsEventOctopus.EventHandlerCallback{
			nutsEventOctopus.EventDistributedConsentRequestReceived: func(event *nutsEventOctopus.Event) {
				cordaChannel.HandleUpdatedEventState(event)
				err := cordaChannel.ReceiveEvent(event)
				if err != nil {
					logger.Logger().Error(err)
					errorDescription := err.Error()
					event.Error = &errorDescription
					if err.Recoverable() {
						cordaChannel.Publish(nutsEventOctopus.ChannelConsentRetry, event)
					} else {
						event.Name = nutsEventOctopus.EventErrored
						cordaChannel.Publish(nutsEventOctopus.ChannelConsentRequest, event)
					}
				}
			},
			nutsEventOctopus.EventConsentRequestConstructed: func(event *nutsEventOctopus.Event) {
				cordaChannel.HandleUpdatedEventState(event)
			},
			nutsEventOctopus.EventConsentRequestValid: func(event *nutsEventOctopus.Event) {
				cordaChannel.HandleUpdatedEventState(event)
				cordaChannel.HandleEventConsentRequestValid(event)
			},
			nutsEventOctopus.EventConsentRequestAcked: func(event *nutsEventOctopus.Event) {
				cordaChannel.HandleUpdatedEventState(event)
				cordaChannel.HandleEventConsentRequestAcked(event)
			} ,
			nutsEventOctopus.EventConsentDistributed: func(event *nutsEventOctopus.Event) {
				cordaChannel.HandleUpdatedEventState(event)
				cordaChannel.HandleEventConsentDistributed(event)
			},
		})
	if err != nil {
		panic(err)
	}

	go func() {
		for e := range eventbus.Errors() {
			logger.Logger().Errorf("[eventbus] %s\n", e.Error())

		}
	}()

	return nil
}

func (cl ConsentService) Shutdown() error {
	return nil
}
