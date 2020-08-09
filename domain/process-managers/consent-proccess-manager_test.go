package process_managers

import (
	"context"
	"github.com/google/uuid"
	"github.com/looplab/eventhorizon"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	"github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	commands3 "github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	commands2 "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	nutsCryto "github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

type testcase struct {
	saga        ConsentProgressManager
	event       eventhorizon.Event
	commands    []eventhorizon.Command
	prepareF    func()
	customTestF func(t *testing.T, tcase *testcase)
}

func TestConsentProgressManager_RunSaga(t *testing.T) {
	consentID := uuid.New()
	knownCustodianID := "agb:123"

	cryptoClient := nutsCryto.NewCryptoClient()
	legalEntity := types.LegalEntity{URI: knownCustodianID}
	entityKey := types.KeyForEntity(legalEntity)
	cryptoClient.GenerateKeyPair(entityKey)

	consentData := events.ConsentData{
		CustodianID: knownCustodianID,
		SubjectID:   "bsn:123",
		ActorID:     "agb:456",
	}
	externalUUID, _ := ConsentProgressManager{}.CalculateExternalUUID(consentData)
	externalID, _ := ConsentProgressManager{}.CalculateExternalID(consentData)

	cases := map[string]testcase{
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
			nil,
		},
		"ConsentRequestRegistered, will try to reserve the consent and create negotiation": {
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
			[]eventhorizon.Command{
				&commands2.ReserveConsent{
					ID:          externalUUID,
					ConsentID:   consentID,
					CustodianID: knownCustodianID,
					SubjectID:   "bsn:123",
					ActorID:     "agb:456",
					Class:       "transfer",
					Start:       time.Time{},
					End:         time.Time{},
				}},
			nil,
			func(t *testing.T, tcase *testcase) {
				commands := tcase.saga.RunSaga(context.Background(), tcase.event)
				if assert.Equal(t, 2, len(commands)) {
					// check existence of CreateNegotiationCommand
					actual := commands[0]

					cmdData, ok := actual.(*commands3.CreateNegotiation)
					if assert.True(t, ok, "command should be of type createNegotiation") {
						assert.Equal(t, cmdData.ExternalNegotiationID, string(externalID))
						assert.Equal(t, cmdData.CustodianID,knownCustodianID)
						assert.Equal(t, cmdData.SubjectID, "bsn:123")
						assert.Equal(t, cmdData.ActorID, "agb:456")
					}

					// check existence of ReserveConsentCommand
					if !reflect.DeepEqual(commands[1], tcase.commands[0]) {
						t.Errorf("test case '%s': incorrect command", commands[1].CommandType())
						t.Logf("exp: %#v\n", tcase.commands[0])
						t.Logf("got: %#v\n", commands[1])
					}
				}
			},
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			if tcase.prepareF != nil {
				tcase.prepareF()
			}
			if tcase.customTestF != nil {
				tcase.customTestF(t, &tcase)
			} else {

				commands := tcase.saga.RunSaga(context.Background(), tcase.event)
				if len(commands) != len(tcase.commands) {
					t.Errorf("test case '%s': incorrect amount of commands", name)
					t.Logf("exp: %#v\n", len(tcase.commands))
					t.Logf("got: %#v\n", len(commands))
				}
				for i, cmd := range tcase.commands {
					if !reflect.DeepEqual(cmd, commands[i]) {
						t.Errorf("test case '%s': incorrect command", cmd.CommandType())
						t.Logf("exp: %#v\n", cmd)
						t.Logf("got: %#v\n", commands[i])
					}
				}
			}
		})
	}
}
