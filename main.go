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
	"github.com/nuts-foundation/nuts-consent-service/domain/consent"
	"log"
	"time"
)

func main() {
	println("nuts consent service")

	eventLogger := &EventLogger{}

	eh.RegisterAggregate(func(id uuid.UUID) eh.Aggregate {
		return &consent.ConsentAggregate{
			AggregateBase: events.NewAggregateBase(consent.ConsentAggregateType, id),
		}
	})

	eventstore := memory.NewEventStore()
	eventbus := local.NewEventBus(local.NewGroup())
	eventbus.AddObserver(eh.MatchAny(), eventLogger)
	aggregateStore, err := events.NewAggregateStore(eventstore, eventbus)
	if err != nil {
		log.Fatal(err)
	}

	commandBus := bus.NewCommandHandler()
	consentCommandHandler, err := aggregate.NewCommandHandler(consent.ConsentAggregateType, aggregateStore)
	if err != nil {
		log.Fatal(err)
	}

	commandHandler := eh.UseCommandHandlerMiddleware(consentCommandHandler, eventLogger.CommandLogger)
	commandBus.SetHandler(commandHandler, consent.ProposeCmdType)
	commandBus.SetHandler(commandHandler, consent.CancelCmdType)


	uniquenessSaga := saga.NewEventHandler(&consent.UniquenessSaga{}, commandBus)
	eventbus.AddHandler(eh.MatchEvent(consent.Proposed), uniquenessSaga)

	id := uuid.New()

	proposeConsentCmd := &consent.Propose{
		ID:          id,
		CustodianID: "agb:123",
		SubjectID:   "bsn:999",
		ActorID:     "agb:456",
		Start:       time.Now(),
	}

	err = commandHandler.HandleCommand(context.Background(), proposeConsentCmd)
	if err != nil {
		panic(err)
	}

	proposeConsentCmd.ID = uuid.New()

	err = commandHandler.HandleCommand(context.Background(), proposeConsentCmd)
	if err != nil {
		panic(err)
	}
	//repo := memory2.NewRepo()
	//crRepo := version.NewRepo(repo)

	//crProjector := projector.NewEventHandler(domain.Projector{}, crRepo)
	//crProjector.SetEntityFactory(func() eh.Entity { return &domain.ConsentRequest{} })
	//eventbus.AddHandler(eh.MatchAnyEventOf(domain.Initiated, domain.Received), crProjector)


	time.Sleep(100*time.Millisecond)

	println("end")
}
