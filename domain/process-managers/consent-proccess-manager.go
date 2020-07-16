package process_managers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	consent_utils "github.com/nuts-foundation/nuts-consent-service/consent-utils"
	"github.com/nuts-foundation/nuts-consent-service/domain"
	consentCommands "github.com/nuts-foundation/nuts-consent-service/domain/consent/commands"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	treatmentRelationCommands "github.com/nuts-foundation/nuts-consent-service/domain/treatment-relation/commands"
	nutsCryto "github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	"log"
)

const ConsentProgressManagerType = saga.Type("consentProgressManager")

// ConsentProgressManager manges the process of registering and synchronising a consent
// It sits between the consent aggregate and the treatment relation aggregate.
// This process manager decouples the two aggregates.
type ConsentProgressManager struct {
}

func (c ConsentProgressManager) SagaType() saga.Type {
	return ConsentProgressManagerType
}

func (c ConsentProgressManager) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	log.Printf("[ConsentProsessManager] event: %+v\n", event)
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
			log.Println("[ConsentProsessManager] could not cast data from ReservationAccepted event")
		}
		utils := consent_utils.ConsentUtils{}

		var fhirConsent string
		var err error

		if fhirConsent, err = utils.CreateFhirConsentResource(data); err != nil {
			return []eh.Command{
				&consentCommands.RejectConsentRequest{
					ID:     data.ID,
					Reason: fmt.Sprintf("[ConsentProsessManager] could not create the FHIR consent resource: %w", err),
				},
			}
		}
		log.Printf("[ConsentProsessManager] FHIR resource created: %s\n", fhirConsent)
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
