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
		consentRequestData, ok := event.Data().(events.ConsentData)
		if !ok {
			logger.Logger().Errorf("[ConsentProsessManager] could not cast consentRequestData from ConsentRequestRegistered event")
			return nil
		}

		cryptoClient := nutsCryto.NewCryptoClient()
		legalEntity := types.LegalEntity{URI: consentRequestData.CustodianID}
		entityKey := types.KeyForEntity(legalEntity)
		custodianCheck := cryptoClient.PrivateKeyExists(entityKey)

		if !custodianCheck {
			return []eh.Command{
				&consentCommands.RejectConsentRequest{
					ID:     consentRequestData.ID,
					Reason: "Custodian is not managed by this node",
				},
			}
		}

		treatmentID, err := c.CalculateExternalID(consentRequestData)
		if err != nil {
			return []eh.Command{
				&consentCommands.RejectConsentRequest{
					ID:     consentRequestData.ID,
					Reason: fmt.Sprintf("Could not generate treatmentID: %s", err),
				},
			}
		}

		return []eh.Command{
			&treatmentRelationCommands.ReserveConsent{
				ID: treatmentID,

				ConsentID:   consentRequestData.ID,
				CustodianID: consentRequestData.CustodianID,
				SubjectID:   consentRequestData.SubjectID,
				ActorID:     consentRequestData.ActorID,
				Class:       consentRequestData.Class,
				Start:       consentRequestData.Start,
				End:         consentRequestData.End,
			},
		}
	case events.ReservationAccepted:
		consentData, ok := event.Data().(events.ConsentData)
		if !ok {
			logger.Logger().Errorf("[ConsentProsessManager] could not cast consentRequestData from ReservationAccepted event")
			return nil
		}
		return []eh.Command{
			&commands.PrepareNegotiation{
				ID:          uuid.New(),
				ConsentData: consentData,
			},
		}
	case events.ReservationRejected:
		return []eh.Command{
			&consentCommands.RejectConsentRequest{
				ID:     uuid.UUID{},
				Reason: "ConsentRequest already exists for this combination of custodian, actor, subject, class and period",
			},
		}
	case events.NegotiationPrepared:
		//consentRequestData, ok := event.Data().(events.NegotiationData)
		//if !ok {
		//	pkg.Logger().Tracef("[ConsentProsessManager] could not cast consentRequestData from NegotiationPrepared event")
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
