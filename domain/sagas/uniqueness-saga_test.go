package sagas

import (
	"context"
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"reflect"
	"testing"
)

func TestUniquenessSaga_RunSaga(t *testing.T) {
	id := uuid.New()
	proposedData := events.ProposedData{
		ID:          id,
		CustodianID: "agb:123",
		SubjectID:   "bsn:999",
		ActorID:     "agb:456",
		Start:       consent.TimeNow(),
	}

	uniqeID := proposedData.CustodianID + proposedData.SubjectID + proposedData.ActorID

	cases := map[string]struct {
		saga     UniquenessSaga
		event    eventhorizon.Event
		commands []eventhorizon.Command
	}{
		"first time": {
			UniquenessSaga{existingIds: make([]string, 0)},
			eventhorizon.NewEventForAggregate(events.Proposed, proposedData, consent.TimeNow(), domain.ConsentAggregateType, id, 1),
			[]eventhorizon.Command{&commands.MarkAsUnique{ID: id}},
		},
		"duplicate": {
			UniquenessSaga{existingIds: []string{uniqeID}},
			eventhorizon.NewEventForAggregate(events.Proposed, proposedData, consent.TimeNow(), domain.ConsentAggregateType, id, 1),
			[]eventhorizon.Command{&commands.Cancel{
				ID:     id,
				Reason: "duplicate consent",
			}},
		},
	}

	for name, testcase := range cases {
		t.Run(name, func(t *testing.T) {
			commands := testcase.saga.RunSaga(context.Background(), testcase.event)
			if !reflect.DeepEqual(commands, testcase.commands) {
				t.Errorf("test case '%s': incorrect commands", name)
				t.Logf("exp: %#v\n", testcase.commands)
				t.Logf("got: %#v\n", commands)
			}
		})

	}
}
