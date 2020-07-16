package treatment_relation

import (
	"context"
	"errors"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	domainEvents "github.com/nuts-foundation/nuts-consent-service/domain/events"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	"reflect"
	"testing"
	"time"
)

func TestTreatmentRelationAggregate_HandleCommand(t *testing.T) {
	TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.Local)
	}

	id := uuid.New()
	cases := map[string]struct {
		agg            *TreatmentRelationAggregate
		cmd            eh.Command
		expectedEvents []eh.Event
		expectedError  error
	}{
		"unknown command": {
			&TreatmentRelationAggregate{
				AggregateBase: events.NewAggregateBase(domain.ConsentAggregateType, id),
			},
			&mocks.Command{
				ID:      id,
				Content: "testcontent of unknown command",
			},
			nil,
			domain.ErrUnknownCommand,
		},
		"reserve consent": {
			&TreatmentRelationAggregate{
				AggregateBase: events.NewAggregateBase(domain.ConsentAggregateType, id),
			},
			&treatmentRelationCommands.ReserveConsent{
				ID:          id,
				CustodianID: "agb:123",
				SubjectID:   "bsn:999",
				ActorID:     "agb:456",
				Start:       TimeNow(),
			}, []eh.Event{eh.NewEventForAggregate(domainEvents.ReservationAccepted, domainEvents.ConsentData{
				ID:          id,
				CustodianID: "agb:123",
				SubjectID:   "bsn:999",
				ActorID:     "agb:456",
				Start:       TimeNow(),
			}, TimeNow(), domain.ConsentAggregateType, id, 1)}, nil,
		},
	}

	for name, testcase := range cases {
		t.Run(name, func(t *testing.T) {
			//t.Parallel()
			err := testcase.agg.HandleCommand(context.Background(), testcase.cmd)
			if (testcase.expectedError != nil && err == nil) ||
				(testcase.expectedError == nil && err != nil) ||
				(testcase.expectedError != nil && err != nil && !(err.Error() == testcase.expectedError.Error() || errors.Is(err, testcase.expectedError))) {
				t.Errorf("incorrect error result")
				t.Log("exp error: ", testcase.expectedError)
				t.Log("got error: ", err)
			}

			events := testcase.agg.Events()
			if !reflect.DeepEqual(events, testcase.expectedEvents) {
				t.Errorf("test case '%s': incorrect events", name)
				t.Logf("exp: %#v\n", testcase.expectedEvents)
				t.Logf("got: %#v\n", events)
			}
		})

	}
}
