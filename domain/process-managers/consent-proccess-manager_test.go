package process_managers

import (
	"context"
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	commands2 "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	nutsCryto "github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	"reflect"
	"testing"
	"time"
)

func TestConsentProgressManager_RunSaga(t *testing.T) {
	consentID := uuid.New()
	knownCustodianID := "agb:123"

	cryptoClient := nutsCryto.NewCryptoClient()
	legalEntity := types.LegalEntity{URI: knownCustodianID}
	entityKey := types.KeyForEntity(legalEntity)
	cryptoClient.GenerateKeyPair(entityKey)

	externalID, _ := ConsentProgressManager{}.CalculateExternalID(events.ConsentData{
		CustodianID: knownCustodianID,
		SubjectID:   "bsn:123",
		ActorID:     "agb:456",
	})

	cases := map[string]struct {
		saga     ConsentProgressManager
		event    eventhorizon.Event
		commands []eventhorizon.Command
		prepareF func()
	}{
		"ConsentRegistered, custodian not managed rejects request": {
			ConsentProgressManager{},
			eventhorizon.NewEventForAggregate(events.ConsentRequestRegistered, events.ConsentData{
				ID:          consentID,
				CustodianID: "",
				SubjectID:   "",
				ActorID:     "",
				Class:       "",
				Start:       time.Time{},
				End:         time.Time{},
			}, time.Now(), domain.ConsentAggregateType, consentID, 1),
			[]eventhorizon.Command{&commands.RejectConsentRequest{
				ID:     consentID,
				Reason: "Custodian is not managed by this node",
			}},
			nil,
		},
		"ConsentRegistered, known custodian will start sync": {
			ConsentProgressManager{},
			eventhorizon.NewEventForAggregate(events.ConsentRequestRegistered, events.ConsentData{
				ID:          consentID,
				CustodianID: knownCustodianID,
				SubjectID:   "bsn:123",
				ActorID:     "agb:456",
				Class:       "transfer",
				Start:       time.Time{},
				End:         time.Time{},
			}, time.Now(), domain.ConsentAggregateType, consentID, 1),
			[]eventhorizon.Command{&commands2.ReserveConsent{
				ID:          externalID,
				CustodianID: knownCustodianID,
				SubjectID:   "bsn:123",
				ActorID:     "agb:456",
				Class:       "transfer",
				Start:       time.Time{},
				End:         time.Time{},
			}},
			func() {
			},
		}}

	for name, testcase := range cases {
		t.Run(name, func(t *testing.T) {
			if testcase.prepareF != nil {
				testcase.prepareF()
			}
			commands := testcase.saga.RunSaga(context.Background(), testcase.event)
			if len(commands) != len(testcase.commands) {
				t.Errorf("test case '%s': incorrect amount of commands", name)
				t.Logf("exp: %#v\n", len(testcase.commands))
				t.Logf("got: %#v\n", len(commands))
			}
			for i, cmd := range testcase.commands {
				if !reflect.DeepEqual(cmd, commands[i]) {
					t.Errorf("test case '%s': incorrect command", name)
					t.Logf("exp: %#v\n", cmd)
					t.Logf("got: %#v\n", commands[i])
				}
			}
		})
	}
}