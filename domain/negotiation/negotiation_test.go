package negotiation

import (
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/nuts-foundation/consent-bridge-go-client/api"
	consent_utils "github.com/nuts-foundation/nuts-consent-service/consent-utils"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	core "github.com/nuts-foundation/nuts-go-core"
	"reflect"
	"testing"
	"time"
)

type mockChannel struct {
}

func (j mockChannel) StartSync(eventID uuid.UUID, externalID string, initiatingPartyID string, consentFacts []consent_utils.ConsentFact) error {
	return nil
}

func (j mockChannel) BuildFullConsentRequestState(eventID uuid.UUID, externalID string, consentFacts []consent_utils.ConsentFact) (api.FullConsentRequestState, error) {
	return api.FullConsentRequestState{}, nil
}

func (j mockChannel) ReceiveEvent(event interface{}) core.Error {
	panic("implement me")
}

func (j mockChannel) Publish(subject string, event interface{}) error {
	return nil
}

func TestNegotiationAggregate_HandleCommand(t *testing.T) {
	id := uuid.New()
	consentId := uuid.New()
	consentData := events2.ConsentData{
		ID:          consentId,
		CustodianID: "123",
		SubjectID:   "999",
		ActorID:     "456",
		Class:       "medical",
		Start:       TimeNow(),
		End:         time.Time{},
	}

	consentFact, _ := consent_utils.MockConsentFactBuilder{}.BuildFact(consentData)

	TimeNow = func() time.Time {
		return time.Date(2017, time.July, 10, 23, 0, 0, 0, time.Local)
	}

	cases := map[string]struct {
		agg            *NegotiationAggregate
		cmd            eh.Command
		expectedEvents []eh.Event
		expectedError  error
		setup          func(t *testing.T, aggregate *NegotiationAggregate, ctrl *gomock.Controller)
	}{
		"err - unknown command": {
			agg: &NegotiationAggregate{
				AggregateBase: events.NewAggregateBase(domain.ConsentNegotiationAggregateType, id),
			},
			cmd: &mocks.Command{
				id,
				"test content",
			},
			expectedEvents: nil,
			expectedError:  domain.ErrUnknownCommand,
		},
		"ok - create negotiation": {
			agg: &NegotiationAggregate{
				AggregateBase: events.NewAggregateBase(domain.ConsentNegotiationAggregateType, id),
			},
			cmd: &commands.CreateNegotiation{
				ID:          id,
				CustodianID: "123",
				ActorID:     "456",
				SubjectID:   "999",
			},
			expectedEvents: []eh.Event{eh.NewEventForAggregate(events2.NegotiationCreated, events2.NegotiationBaseData{
				CustodianID: "123",
				SubjectID:   "999",
				ActorID:     "456",
			}, TimeNow(), domain.ConsentNegotiationAggregateType, id, 1)},
			expectedError: nil,
		},
		"ok - update channel state": {
			agg: &NegotiationAggregate{
				AggregateBase: events.NewAggregateBase(domain.ConsentNegotiationAggregateType, id),
			},
			cmd: &commands.UpdateState{
				ID:    id,
				State: "new state",
			},
			expectedEvents: []eh.Event{eh.NewEventForAggregate(events2.NegotiationStateUpdated, events2.ChannelStateData{State: "new state"}, TimeNow(), domain.ConsentNegotiationAggregateType, id, 1)},
			expectedError:  nil,
		},
		"ok - propose consent": {
			agg: &NegotiationAggregate{
				AggregateBase: events.NewAggregateBase(domain.ConsentNegotiationAggregateType, id),
				Channel:       mockChannel{},
				ConsentFacts:  [][]byte{consentFact},
				FactBuilder:   consent_utils.MockConsentFactBuilder{},
				subjectID:     consentData.SubjectID,
				custodianID:   consentData.CustodianID,
				actorID:       consentData.ActorID,
			},
			cmd: &commands.ProposeConsent{
				ID: id,
			},
			expectedEvents: []eh.Event{eh.NewEventForAggregate(events2.ConsentProposed, nil, TimeNow(), domain.ConsentNegotiationAggregateType, id, 1)},
			expectedError:  nil,
		},
	}

	for name, testcase := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			//t.Parallel()
			if testcase.setup != nil {
				testcase.setup(t, testcase.agg, ctrl)
			}
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

			ctrl.Finish()
		})
	}
}
