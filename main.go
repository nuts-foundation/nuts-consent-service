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
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent"
	consentCommands "github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	process_managers "github.com/nuts-foundation/nuts-consent-service/domain/process-managers"
	treatment_relation "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	"github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	"log"
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
	if err != nil {
		log.Fatal(err)
	}

	treatmentCommandHander, err := aggregate.NewCommandHandler(domain.TreatmentRelationAggregateType, aggregateStore)

	//negotiationCommandHandler, err := aggregate.NewCommandHandler(negotiation.ConsentNegotiationAggregateType, aggregateStore)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//consentCommandHandler = eh.UseCommandHandlerMiddleware(consentCommandHandler, eventLogger.CommandLogger)
	//negotiationCommandHandler = eh.UseCommandHandlerMiddleware(negotiationCommandHandler, eventLogger.CommandLogger)
	commandBus.SetHandler(consentCommandHandler, consentCommands.RegisterConsentCmdType)
	commandBus.SetHandler(treatmentCommandHander, treatmentRelationCommands.ReserveConsentCmdType)
	//commandBus.SetHandler(consentCommandHandler, commands.CancelCmdType)
	//commandBus.SetHandler(consentCommandHandler, commands.MarkAsErroredCmdType)
	//commandBus.SetHandler(consentCommandHandler, commands.MarkAsUniqueCmdType)
	//commandBus.SetHandler(consentCommandHandler, commands.StartSyncCmdType)
	//commandBus.SetHandler(consentCommandHandler, commands.MarkCustodianCheckedCmdType)

	consentProgressManager := saga.NewEventHandler(process_managers.ConsentProgressManager{}, commandBus)
	eventbus.AddHandler(eh.MatchEvent(events2.ConsentRequestRegistered), consentProgressManager)
	//uniquenessSaga := saga.NewEventHandler(sagas.NewUniquenessSaga(), commandBus)
	//eventbus.AddHandler(eh.MatchEvent(events2.Proposed), uniquenessSaga)

	//negotiationRepo := version.NewRepo(memory2.NewRepo())
	//projector := projector2.NewEventHandler(&consent.SyncProjector{}, negotiationRepo)
	//projector.SetEntityFactory(func() eh.Entity { return &consent.ConsentNegotiation{} })
	//eventbus.AddHandler(eh.MatchAny(), projector)

	//syncSaga := saga.NewEventHandler(sagas.SyncSaga{NegotiationRepo: negotiationRepo}, commandBus)
	//eventbus.AddHandler(eh.MatchAnyEventOf(events2.Unique), syncSaga)

	//checkPartiesSaga := saga.NewEventHandler(sagas.CheckPartiesSaga{}, commandBus)
	//eventbus.AddHandler(eh.MatchAnyEventOf(events2.Proposed), checkPartiesSaga)

	id := uuid.New()

	// make sure the custodian has a keypair in the truststore
	crypto := pkg.NewCryptoClient()
	custodianID := "agb:123"
	keyID := types.KeyForEntity(types.LegalEntity{custodianID})
	crypto.GenerateKeyPair(keyID)

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
