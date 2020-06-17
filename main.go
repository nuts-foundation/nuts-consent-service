package main

import (
	"context"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/aggregate"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	"github.com/looplab/eventhorizon/eventbus/local"
	projector2 "github.com/looplab/eventhorizon/eventhandler/projector"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/looplab/eventhorizon/eventstore/memory"
	memory2 "github.com/looplab/eventhorizon/repo/memory"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/sagas"
	"log"
	"time"
)

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

	consentCommandHandler, err := aggregate.NewCommandHandler(consent.ConsentAggregateType, aggregateStore)
	if err != nil {
		log.Fatal(err)
	}

	//negotiationCommandHandler, err := aggregate.NewCommandHandler(negotiation.ConsentNegotiationAggregateType, aggregateStore)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//consentCommandHandler = eh.UseCommandHandlerMiddleware(consentCommandHandler, eventLogger.CommandLogger)
	//negotiationCommandHandler = eh.UseCommandHandlerMiddleware(negotiationCommandHandler, eventLogger.CommandLogger)
	if err := commandBus.SetHandler(consentCommandHandler, consent.ProposeCmdType); err != nil {
		panic(err)
	}
	if err := commandBus.SetHandler(consentCommandHandler, consent.CancelCmdType); err != nil {
		panic(err)
	}
	if err := commandBus.SetHandler(consentCommandHandler, consent.MarkAsUniqueCmdType); err != nil {
		panic(err)
	}

	uniquenessSaga := saga.NewEventHandler(sagas.NewUniquenessSaga(), commandBus)
	eventbus.AddHandler(eh.MatchEvent(events2.Proposed), uniquenessSaga)

	negotiationRepo := memory2.NewRepo()
	projector := projector2.NewEventHandler(&consent.SyncProjector{}, negotiationRepo)
	projector.SetEntityFactory(func() eh.Entity { return &consent.ConsentNegotiation{} })
	eventbus.AddHandler(eh.MatchAny(), projector)

	syncSaga := saga.NewEventHandler(sagas.SyncSaga{NegotiationRepo: negotiationRepo}, commandBus)
	eventbus.AddHandler(eh.MatchAnyEventOf(events2.Proposed, events2.Unique), syncSaga)

	id := uuid.New()

	proposeConsentCmd := &consent.Propose{
		ID:          id,
		CustodianID: "agb:123",
		SubjectID:   "bsn:999",
		ActorID:     "agb:456",
		Start:       time.Now(),
	}

	err = commandBus.HandleCommand(context.Background(), proposeConsentCmd)

	//proposeConsentCmd.ID = uuid.New()
	//err = commandBus.HandleCommand(context.Background(), proposeConsentCmd)

	for {
		log.Println(<-eventbus.Errors())
	}

	println("end")
}
