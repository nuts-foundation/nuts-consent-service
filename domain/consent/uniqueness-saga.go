package consent

import (
	"context"
	"fmt"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
)

type UniquenessSaga struct {
	existingIds []string
}

func NewUniquenessSaga() *UniquenessSaga {
	return &UniquenessSaga{existingIds: make([]string, 0)}
}

const UniquenessSagaType saga.Type = "ConsentUniquenessSaga"

func (s UniquenessSaga) SagaType() saga.Type {
	return UniquenessSagaType
}

func (s *UniquenessSaga) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	fmt.Printf("[%s] event received: %+v\n", UniquenessSagaType, event)
	switch event.EventType() {
	case Proposed:
		data, ok := event.Data().(ProposedData)
		if ok {
			id := data.CustodianID + data.SubjectID + data.ActorID
			fmt.Printf("checking duplicates for: %v\n", id)
			for _, existingId := range s.existingIds {
				fmt.Printf("against known id: %v\n", existingId)
				if id == existingId {
					fmt.Println("duplicate found!")
					return []eh.Command{Cancel{
						ID:     event.AggregateID(),
						Reason: "duplicate consent",
					}}
				}
			}
			s.existingIds = append(s.existingIds, id)
		}

	}
	return nil
}
