package pkg

import (
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
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	process_managers "github.com/nuts-foundation/nuts-consent-service/domain/process-managers"
	treatment_relation "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	nutsConsentStoreClient "github.com/nuts-foundation/nuts-consent-store/client"
	nutsConsentStorePkg "github.com/nuts-foundation/nuts-consent-store/pkg"
	nutsCryptoPkg "github.com/nuts-foundation/nuts-crypto/pkg"
	nutsEventOctClient "github.com/nuts-foundation/nuts-event-octopus/client"
	nutsEventOctopus "github.com/nuts-foundation/nuts-event-octopus/pkg"
	core "github.com/nuts-foundation/nuts-go-core"
	"github.com/nuts-foundation/nuts-registry/client"
	"github.com/nuts-foundation/nuts-registry/pkg"
	"log"
	"sync"
)

type ConsentServiceConfig struct {
}

type ConsentServiceClient interface {
	StartConsentFlow(*CreateConsentRequest) (*uuid.UUID, error)
	HandleIncomingCordaEvent(event *nutsEventOctopus.Event)
}

type ConsentService struct {
	NutsRegistry     pkg.RegistryClient
	NutsCrypto       nutsCryptoPkg.Client
	NutsConsentStore nutsConsentStorePkg.ConsentStoreClient
	NutsEventOctopus nutsEventOctopus.EventOctopusClient
	Config           ConsentServiceConfig
	EventPublisher   nutsEventOctopus.IEventPublisher
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
	panic("implement me")
}

func (cl ConsentService) HandleIncomingCordaEvent(event *nutsEventOctopus.Event) {
	panic("implement me")
}

func (cl ConsentService) Configure() error {
	return nil
}

func (cl *ConsentService) Start() error {
	cl.NutsCrypto = nutsCryptoPkg.NewCryptoClient()
	cl.NutsRegistry = client.NewRegistryClient()
	cl.NutsConsentStore = nutsConsentStoreClient.NewConsentStoreClient()
	cl.NutsEventOctopus = nutsEventOctClient.NewEventOctopusClient()
	// This module has no mode feature (server/client) so we delegate it completely to the global mode
	if core.NutsConfig().GetEngineMode("") != core.ServerEngineMode {
		return nil
	}
	publisher, err := cl.NutsEventOctopus.EventPublisher("consent-logic")
	if err != nil {
		logger().WithError(err).Panic("Could not subscribe to event publisher")
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
	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &negotiation.NegotiationAggregate{
			AggregateBase:  events.NewAggregateBase(domain.ConsentNegotiationAggregateType, id),
			FactBuilder:    consent_utils.FhirConsentFact{},
			EventPublisher: publisher,
		}
	})

	eventstore := memory.NewEventStore()
	eventbus := local.NewEventBus(local.NewGroup())
	commandBus := bus.NewCommandHandler()

	eventLogger := &EventLogger{}
	eventbus.AddObserver(eh.MatchAny(), eventLogger)

	aggregateStore, err := events.NewAggregateStore(eventstore, eventbus)
	if err != nil {
		log.Fatal(err)
	}

	consentCommandHandler, err := aggregate.NewCommandHandler(domain.ConsentAggregateType, aggregateStore)
	treatmentCommandHander, err := aggregate.NewCommandHandler(domain.TreatmentRelationAggregateType, aggregateStore)
	negotiationCommandHandler, err := aggregate.NewCommandHandler(domain.ConsentNegotiationAggregateType, aggregateStore)

	commandBus.SetHandler(consentCommandHandler, consentCommands.RegisterConsentCmdType)
	commandBus.SetHandler(treatmentCommandHander, treatmentRelationCommands.ReserveConsentCmdType)
	commandBus.SetHandler(consentCommandHandler, consentCommands.RejectConsentCmdType)
	commandBus.SetHandler(negotiationCommandHandler, commands.PrepareNegotiationCmdType)
	commandBus.SetHandler(negotiationCommandHandler, commands.ProposeConsentFactCmdType)

	consentProgressManager := saga.NewEventHandler(process_managers.ConsentProgressManager{}, commandBus)
	eventbus.AddHandler(eh.MatchAnyEventOf(
		events2.ConsentRequestRegistered,
		events2.ReservationAccepted,
		events2.NegotiationPrepared,
	), consentProgressManager)

	// TODO: handle these in the Negotiation Aggregate
	//err = cl.NutsEventOctopus.Subscribe("consent-logic",
	//	nutsEventOctopus.ChannelConsentRequest,
	//	map[string]nutsEventOctopus.EventHandlerCallback{
	//		nutsEventOctopus.EventDistributedConsentRequestReceived: cl.HandleIncomingCordaEvent,
	//		nutsEventOctopus.EventConsentRequestValid:               cl.HandleEventConsentRequestValid,
	//		nutsEventOctopus.EventConsentRequestAcked:               cl.HandleEventConsentRequestAcked,
	//		nutsEventOctopus.EventConsentDistributed:                cl.HandleEventConsentDistributed,
	//	})
	//if err != nil {
	//	panic(err)
	//}
	return nil
}

func (cl ConsentService) Shutdown() error {
	return nil
}
