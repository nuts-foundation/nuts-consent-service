package consent_utils

import (
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwk"
	bridgeClient "github.com/nuts-foundation/consent-bridge-go-client/api"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	"github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	"github.com/nuts-foundation/nuts-registry/client"
	"time"
)

type SyncChannel interface {
	BuildFullConsentRequestState(eventID uuid.UUID, consentID []byte, consentFact ConsentFact) bridgeClient.FullConsentRequestState
}

type CordaChannel struct {
}

func (c CordaChannel) BuildFullConsentRequestState(eventID uuid.UUID, externalID string, consentFact ConsentFact) (bridgeClient.FullConsentRequestState, error) {
	now := time.Now()

	var records []bridgeClient.ConsentRecord
	record, err := prepareRecord(consentFact)
	if err != nil {
		return bridgeClient.FullConsentRequestState{}, err
	}
	records = append(records, record)

	initiatingLegalEntitiy := consentFact.Custodian()
	// TODO: get this from the config
	// nodeIdentity := core.NutsConfig().Identity()
	nodeIdentity := "urn:oid:1.3.6.1.4.1.54851.4:123"

	legalEntities := []bridgeClient.Identifier{
		bridgeClient.Identifier(consentFact.Actor()),
		bridgeClient.Identifier(consentFact.Custodian()),
	}

	return bridgeClient.FullConsentRequestState{
		Comment:               nil,
		ConsentId:             bridgeClient.ConsentId{ExternalId: &externalID, UUID: eventID.String()},
		ConsentRecords:        records,
		CreatedAt:             &now,
		InitiatingLegalEntity: bridgeClient.Identifier(initiatingLegalEntitiy),
		InitiatingNode:        &nodeIdentity,
		LegalEntities:         legalEntities,
		UpdatedAt:             &now,
	}, nil
}

func prepareRecord(fact ConsentFact) (bridgeClient.ConsentRecord, error) {

	encryptedConsent, err := encryptConsentFact(fact)
	if err != nil {
		return bridgeClient.ConsentRecord{}, err
	}

	cipherText := base64.StdEncoding.EncodeToString(encryptedConsent.CipherText)

	var validTo *time.Time
	end := fact.End()
	if !end.IsZero() {
		validTo = &end
	}

	bridgeMeta := bridgeClient.Metadata{
		Domain: []bridgeClient.Domain{"medical"},
		Period: bridgeClient.Period{
			ValidFrom: fact.Start(),
			ValidTo:   validTo,
		},
		SecureKey: bridgeClient.SymmetricKey{
			Alg: "AES_GCM", //todo: fix hardcoded alg
			Iv:  base64.StdEncoding.EncodeToString(encryptedConsent.Nonce),
		},
		// Fixme: this previousRecordHash should be part of the consentFact put in the FHIR record.
		//PreviousAttachmentHash: record.PreviousRecordhash,
		PreviousAttachmentHash: nil,
		ConsentRecordHash:      fact.Hash(),
	}

	alg := "RSA-OAEP"
	for i := range encryptedConsent.CipherTextKeys {

		// The DoubleEncryptedCipherText encrypts with public key in the order actor, custodian
		var legalEntity bridgeClient.Identifier
		if i == 0 {
			legalEntity = bridgeClient.Identifier(fact.Actor())
		} else {
			legalEntity = bridgeClient.Identifier(fact.Custodian())
		}

		ctBase64 := base64.StdEncoding.EncodeToString(encryptedConsent.CipherTextKeys[i])
		bridgeMeta.OrganisationSecureKeys = append(bridgeMeta.OrganisationSecureKeys, bridgeClient.ASymmetricKey{
			Alg:         &alg,
			CipherText:  &ctBase64,
			LegalEntity: legalEntity,
		})
	}

	return bridgeClient.ConsentRecord{Metadata: &bridgeMeta, CipherText: &cipherText}, nil

}

func encryptConsentFact(consentFact ConsentFact) (types.DoubleEncryptedCipherText, error) {
	// list of PEM encoded pubic keys to encrypt the record
	var partyKeys []jwk.Key

	// get public key for actor
	registryClient := client.NewRegistryClient()
	organization, err := registryClient.OrganizationById(consentFact.Actor())
	if err != nil {
		logger.Logger().Errorf("error while getting public key for actor: %v from registry: %v\n", consentFact.Actor(), err)
		return types.DoubleEncryptedCipherText{}, err
	}

	jwk, err := organization.CurrentPublicKey()
	if err != nil {
		return types.DoubleEncryptedCipherText{}, fmt.Errorf("registry entry for organization %v does not contain a public key", consentFact.Actor())
	}

	partyKeys = append(partyKeys, jwk)

	// get public key for custodian
	cryptoClient := pkg.NewCryptoClient()
	keyIdentifier := types.KeyForEntity(types.LegalEntity{URI: consentFact.Custodian()})
	jwk, err = cryptoClient.GetPublicKeyAsJWK(keyIdentifier)
	if err != nil {
		logger.Logger().Errorf("error while getting public key for custodian: %v from crypto: %v\n", consentFact.Custodian(), err)
		return types.DoubleEncryptedCipherText{}, err
	}
	partyKeys = append(partyKeys, jwk)

	return cryptoClient.EncryptKeyAndPlainText(consentFact.Payload(), partyKeys)
}
