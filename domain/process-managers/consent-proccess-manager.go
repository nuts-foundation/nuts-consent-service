package process_managers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	consentCommands "github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-consent-service/domain/negotiation/commands"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	nutsCryto "github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
)

const ConsentProgressManagerType = saga.Type("consentProgressManager")

// ConsentProgressManager manges the process of registering and synchronising a consent
// It sits between the consent aggregate and the treatment relation aggregate.
// This process manager decouples the two aggregates.
type ConsentProgressManager struct {
	//FactBuilder consent_utils.ConsentFactBuilder
}

func (c ConsentProgressManager) SagaType() saga.Type {
	return ConsentProgressManagerType
}

func (c ConsentProgressManager) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	logger.Logger().Tracef("[ConsentProsessManager] event: %+v\n", event)
	switch event.EventType() {
	case events.ConsentRequestRegistered:
		data, ok := event.Data().(events.ConsentData)
		if !ok {
			return nil
		}

		cryptoClient := nutsCryto.NewCryptoClient()
		legalEntity := types.LegalEntity{URI: data.CustodianID}
		entityKey := types.KeyForEntity(legalEntity)
		custodianCheck := cryptoClient.PrivateKeyExists(entityKey)

		if !custodianCheck {
			return []eh.Command{
				&consentCommands.RejectConsentRequest{
					ID:     data.ID,
					Reason: "Custodian is not managed by this node",
				},
			}
		}

		treatmentID, err := c.CalculateExternalID(data)
		if err != nil {
			return []eh.Command{
				&consentCommands.RejectConsentRequest{
					ID:     data.ID,
					Reason: fmt.Sprintf("Could not generate treatmentID: %s", err),
				},
			}
		}

		return []eh.Command{
			&treatmentRelationCommands.ReserveConsent{
				ID:          treatmentID,
				CustodianID: data.CustodianID,
				SubjectID:   data.SubjectID,
				ActorID:     data.ActorID,
				Class:       data.Class,
				Start:       data.Start,
				End:         data.End,
			},
		}
	case events.ReservationAccepted:
		data, ok := event.Data().(events.ConsentData)
		if !ok {
			logger.Logger().Tracef("[ConsentProsessManager] could not cast data from ReservationAccepted event")
		}
		return []eh.Command{
			&commands.PrepareNegotiation{
				ID:          uuid.New(),
				ConsentData: data,
			},
		}
	case events.NegotiationPrepared:
		//data, ok := event.Data().(events.NegotiationData)
		//if !ok {
		//	pkg.Logger().Tracef("[ConsentProsessManager] could not cast data from NegotiationPrepared event")
		//}

		return []eh.Command{
			&commands.ProposeConsent{
				ID: event.AggregateID(),
			},
		}
	}

	return nil
}

func (c ConsentProgressManager) CalculateExternalID(data events.ConsentData) (uuid.UUID, error) {
	legalEntity := types.LegalEntity{URI: data.CustodianID}
	entityKey := types.KeyForEntity(legalEntity)
	cryptoClient := nutsCryto.NewCryptoClient()
	externalID, err := cryptoClient.CalculateExternalId(data.SubjectID, data.ActorID, entityKey)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.NewSHA1(domain.NutsExternalIDSpace, externalID), nil
}
