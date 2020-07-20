package main

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
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	process_managers "github.com/nuts-foundation/nuts-consent-service/domain/process-managers"
	treatment_relation "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	"github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	"github.com/nuts-foundation/nuts-event-octopus/client"
	"github.com/nuts-foundation/nuts-event-octopus/engine"
	core "github.com/nuts-foundation/nuts-go-core"
	pkg2 "github.com/nuts-foundation/nuts-registry/pkg"
	"github.com/spf13/cobra"
	"log"
	"os"
	"time"
)

func init() {
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
}

func main() {

	println("nuts consent service")

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

	//negotiationCommandHandler, err := aggregate.NewCommandHandler(negotiation.ConsentNegotiationAggregateType, aggregateStore)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//nutsConfig := core.NutsConfig()
	eventOctopusEngine := engine.NewEventOctopusEngine()
	if err := eventOctopusEngine.Configure(); err != nil {
		panic(err)
	}
	if err := eventOctopusEngine.Start(); err != nil {
		panic(err)
	}

	nutsEventOctopus := client.NewEventOctopusClient()
	publisher, err := nutsEventOctopus.EventPublisher("consent-logic")
	if err != nil {
		log.Panicf("could not subscribe to event publisher: %w", err)
	}

	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &negotiation.NegotiationAggregate{
			AggregateBase:  events.NewAggregateBase(domain.ConsentNegotiationAggregateType, id),
			FactBuilder:    consent_utils.FhirConsentFact{},
			EventPublisher: publisher,
		}
	})

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

	// And now run an basic consent request:
	id := uuid.New()

	// make sure the custodian has a keypair in the truststore
	crypto := pkg.NewCryptoClient()
	custodianID := "urn:oid:2.16.840.1.113883.2.4.6.1:123"
	actorID := "urn:oid:2.16.840.1.113883.2.4.6.1:456"
	keyID := types.KeyForEntity(types.LegalEntity{custodianID})
	crypto.GenerateKeyPair(keyID)

	os.Setenv("NUTS_IDENTITY", "oid:123")
	core.NutsConfig().Load(&cobra.Command{})
	registryPath := "./registry"
	r := pkg2.RegistryInstance()
	r.Config.Mode = "server"
	r.Config.Datadir = registryPath
	r.Config.SyncMode = "fs"
	r.Config.OrganisationCertificateValidity = 1
	r.Config.VendorCACertificateValidity = 1
	if err := r.Configure(); err != nil {
		panic(err)
	}

	// Register a vendor
	_, _ = r.RegisterVendor("Test Vendor", "healthcare")

	// Add Organization to registry
	orgName := "Zorggroep Nuts"
	if _, err := r.VendorClaim(actorID, orgName, nil); err != nil {
		//panic(err)
	}

	proposeConsentCmd := &consentCommands.RegisterConsent{
		ID:          id,
		CustodianID: custodianID,
		SubjectID:   "bsn:999",
		ActorID:     "agb:456",
		Class:       "transfer",
		Start:       time.Now(),
	}

	err = consentCommandHandler.HandleCommand(context.Background(), proposeConsentCmd)
	if err != nil {
		log.Printf("[main] unable to handle command: %s\n", err)
	}

	//proposeConsentCmd.ID = uuid.New()
	//err = commandBus.HandleCommand(context.Background(), proposeConsentCmd)

	go func() {
		for e := range eventbus.Errors() {
			log.Printf("[eventbus] %s\n", e.Error())
		}
	}()

	time.Sleep(5 * time.Second)

	println("end")
}
