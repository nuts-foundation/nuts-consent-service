package consent

import (
	"context"
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"reflect"
	"testing"
)

func TestUniquenessSaga_RunSaga(t *testing.T) {
	id := uuid.New()
	proposedData := ProposedData{
		ID:          id,
		CustodianID: "agb:123",
		SubjectID:   "bsn:999",
		ActorID:     "agb:456",
		Start:       TimeNow(),
	}

	uniqeID := proposedData.CustodianID + proposedData.SubjectID + proposedData.ActorID

	cases := map[string]struct {
		saga     UniquenessSaga
		event    eventhorizon.Event
		commands []eventhorizon.Command
	}{
		"first time": {
			UniquenessSaga{existingIds: make([]string, 0)},
			eventhorizon.NewEventForAggregate(Proposed, proposedData, TimeNow(), ConsentAggregateType, id, 1),
			nil,
		},
		"duplicate": {
			UniquenessSaga{existingIds: []string{uniqeID}},
			eventhorizon.NewEventForAggregate(Proposed, proposedData, TimeNow(), ConsentAggregateType, id, 1),
			[]eventhorizon.Command{Cancel{
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
