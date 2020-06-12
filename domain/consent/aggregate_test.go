package consent

import (
	"context"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"reflect"
	"testing"
	"time"
)

func TestConsentRequestAggregate_HandleCommand(t *testing.T) {
	TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.Local)
	}

	id := uuid.New()
	cases := map[string]struct {
		agg            *ConsentAggregate
		cmd            eh.Command
		expectedEvents []eh.Event
		expectedError  error
	}{
		"unknown command": {
			&ConsentAggregate{
				AggregateBase: events.NewAggregateBase(ConsentAggregateType, id),
			},
			&mocks.Command{
				ID:      id,
				Content: "testcontent of unknown command",
			},
			nil,
			domain.ErrUnknownCommand,
		},
		"propose consent": {
			&ConsentAggregate{
				AggregateBase: events.NewAggregateBase(ConsentAggregateType, id),
			},
			&Propose{
				ID:          id,
				CustodianID: "agb:123",
				SubjectID:   "bsn:999",
				ActorID:     "agb:456",
				Start:       TimeNow(),
			}, []eh.Event{eh.NewEventForAggregate(Proposed, ProposedData{
				ID:          id,
				CustodianID: "agb:123",
				SubjectID:   "bsn:999",
				ActorID:     "agb:456",
				Start:       TimeNow(),
			},TimeNow(), ConsentAggregateType, id, 1)}, nil,
		},
	}

	for name, testcase := range cases {
		t.Run(name, func(t *testing.T) {
			//t.Parallel()
			err := testcase.agg.HandleCommand(context.Background(), testcase.cmd)
			if (testcase.expectedError != nil && err == nil) ||
				(testcase.expectedError == nil && err != nil) ||
				(testcase.expectedError != nil && err != nil && err.Error() != testcase.expectedError.Error()) {
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
