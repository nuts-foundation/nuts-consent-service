package consent_utils

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwk"
	bridgeClient "github.com/nuts-foundation/consent-bridge-go-client/api"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
	cStore "github.com/nuts-foundation/nuts-consent-store/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/cert"
	cryptoTypes "github.com/nuts-foundation/nuts-crypto/pkg/types"
	events "github.com/nuts-foundation/nuts-event-octopus/pkg"
	fhirValidator "github.com/nuts-foundation/nuts-fhir-validation/pkg"
	core "github.com/nuts-foundation/nuts-go-core"
	"github.com/nuts-foundation/nuts-registry/client"
	pkg2 "github.com/nuts-foundation/nuts-registry/pkg"
	"github.com/sirupsen/logrus"
	"github.com/thedevsaddam/gojsonq/v2"
	"time"
)

type SyncChannel interface {
	BuildFullConsentRequestState(eventID uuid.UUID, externalID string, consentFact ConsentFact) (bridgeClient.FullConsentRequestState, error)
	ReceiveEvent(event interface{})
}

type CordaChannel struct {
	Registry   pkg2.RegistryClient
	NutsCrypto pkg.Client
	Publisher  events.IEventPublisher
}

func identity() string {
	return core.NutsConfig().Identity()
}

func (c CordaChannel) Publish(eventName string, event *events.Event) error {
	c.logger().Debugf("Publishing corda event %+v\n", event)
	return c.Publisher.Publish(eventName, *event)
}

func (c CordaChannel) logger() *logrus.Entry {
	return logger.Logger()
}

func (c CordaChannel) ReceiveEvent(event interface{}) {
	cordaEvent := event.(*events.Event)
	crs := bridgeClient.FullConsentRequestState{}
	decodedPayload, err := base64.StdEncoding.DecodeString(cordaEvent.Payload)
	if err != nil {
		errorDescription := fmt.Sprintf("%s: could not base64 decode cordaEvent payload", identity())
		cordaEvent.Error = &errorDescription
		cordaEvent.Name = events.EventErrored
		c.logger().WithError(err).Error(errorDescription)
		_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
	}
	if err := json.Unmarshal(decodedPayload, &crs); err != nil {
		// have cordaEvent-octopus handle redelivery or cancellation
		errorDescription := fmt.Sprintf("%s: could not unmarshall cordaEvent payload", identity())
		cordaEvent.Error = &errorDescription
		cordaEvent.Name = events.EventErrored
		c.logger().WithError(err).Error(errorDescription)
		_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
		return
	}

	// check if all parties signed all attachments, than this request can be finalized by the initiator
	allSigned := true
	for _, cr := range crs.ConsentRecords {
		if cr.Signatures == nil || len(*cr.Signatures) != len(crs.LegalEntities) {
			allSigned = false
		}
	}

	if allSigned {
		c.logger().Debugf("All signatures present for UUID: %s", cordaEvent.ConsentID)
		// Is this node the initiator? InitiatorLegalEntity is only set at the initiating node.
		if cordaEvent.InitiatorLegalEntity != "" {

			// Now check the public keys used by the signatures
			for _, cr := range crs.ConsentRecords {
				for _, signature := range *cr.Signatures {
					// Get the published public key from register
					legalEntityID := signature.LegalEntity
					legalEntity, err := c.Registry.OrganizationById(string(legalEntityID))
					if err != nil {
						errorMsg := fmt.Sprintf("Could not get organization public key for: %s, err: %v", legalEntityID, err)
						cordaEvent.Error = &errorMsg
						c.logger().Debug(errorMsg)
						_ = c.Publish(events.ChannelConsentRetry, cordaEvent)
						return
					}

					jwkFromSig, err := cert.MapToJwk(signature.Signature.PublicKey.AdditionalProperties)
					if err != nil {
						errorMsg := fmt.Sprintf("%s: unable to parse signature public key as JWK: %v", identity(), err)
						c.logger().Warn(errorMsg)
						c.logger().Debugf("publicKey from signature: %s ", signature.Signature.PublicKey)
						cordaEvent.Name = events.EventErrored
						cordaEvent.Error = &errorMsg
						_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
						return
					}

					// Check if the organization owns the public key used for signing and whether it was valid at the moment of signing.
					// ========================
					// TODO: Checking it against the current time is wrong; it should be the time of signing.
					// In practice this won't cause problems for now since certificates used for signing consent records
					// are valid for 1 year since they were introduced (april 2020). So we just have to make sure we
					// switch to a signature format (JWS) which does contain the time of signing before april 2021.
					// https://github.com/nuts-foundation/nuts-consent-logic/issues/45
					checkTime := time.Now()
					orgHasKey, err := legalEntity.HasKey(jwkFromSig, checkTime)
					// Fixme: this error handling should be rewritten
					if err != nil {
						errorMsg := fmt.Sprintf("%s: could not check JWK against organization keys: %v", identity(), err)
						c.logger().Warn(errorMsg)
						cordaEvent.Name = events.EventErrored
						cordaEvent.Error = &errorMsg
						_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
						return
					}

					if !orgHasKey {
						errorMsg := fmt.Sprintf("%s:  organization %s did not have a valid signature for the corresponding public key at the given time %s", core.NutsConfig().Identity(), legalEntityID, checkTime.String())
						c.logger().Warn(errorMsg)
						cordaEvent.Name = events.EventErrored
						cordaEvent.Error = &errorMsg
						_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
						return
					}

					// checking the actual signature here is not required since it's already checked by the CordApp.
				}
			}

			c.logger().Debugf("Sending FinalizeRequest to bridge for UUID: %s", cordaEvent.ConsentID)
			cordaEvent.Name = events.EventAllSignaturesPresent
			_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
		} else {
			c.logger().Debug("This node is not the initiator. Lets wait for the initiator to broadcast EventAllSignaturesPresent")
		}
		return
	}

	c.logger().Debugf("Handling ConsentRequestState: %+v", crs)

	for _, cr := range crs.ConsentRecords {
		// find out which legal entity is ours and still needs signing? It can be more than one, but always take first one missing.
		legalEntityToSignFor := c.findFirstEntityToSignFor(cr.Signatures, crs.LegalEntities)

		// is there work for us?
		if legalEntityToSignFor == "" {
			// nothing to sign for this node/record.
			continue
		}

		// decrypt
		// =======
		fhirConsent, err := c.decryptConsentRecord(cr, legalEntityToSignFor)
		if err != nil {
			errorDescription := fmt.Sprintf("%s: could not decrypt consent record", identity())
			cordaEvent.Name = events.EventErrored
			cordaEvent.Error = &errorDescription
			c.logger().WithError(err).Error(errorDescription)
			_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
			return
		}

		// validate consent record
		// =======================
		factBuilder := FhirConsentFactBuilder{}
		fhirConsentFact, _ := factBuilder.FactFromBytes([]byte(fhirConsent))
		if validationResult, err := factBuilder.VerifyFact(fhirConsentFact.Payload()); !validationResult || err != nil {
			errorDescription := fmt.Sprintf("%s: consent record invalid", identity())
			cordaEvent.Name = events.EventErrored
			cordaEvent.Error = &errorDescription
			c.logger().WithError(err).Error(errorDescription)
			_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
			return
		}

		// publish EventConsentRequestValid
		// ===========================
		cordaEvent.Name = events.EventConsentRequestValid
		_ = c.Publish(events.ChannelConsentRequest, cordaEvent)
	}
}

func (c CordaChannel) decryptConsentRecord(cr bridgeClient.ConsentRecord, legalEntity string) (string, error) {
	encodedCipherText := cr.CipherText
	cipherText, err := base64.StdEncoding.DecodeString(*encodedCipherText)
	// convert hex string of attachment to bytes
	if err != nil {
		return "", err
	}

	if cr.Metadata == nil {
		err := errors.New("missing metadata in consentRequest")
		c.logger().Error(err)
		return "", err
	}

	var encodedLegalEntityKey string
	for _, value := range cr.Metadata.OrganisationSecureKeys {
		if value.LegalEntity == bridgeClient.Identifier(legalEntity) {
			encodedLegalEntityKey = *value.CipherText
		}
	}

	if encodedLegalEntityKey == "" {
		return "", fmt.Errorf("no key found for legalEntity: %s", legalEntity)
	}
	legalEntityKey, _ := base64.StdEncoding.DecodeString(encodedLegalEntityKey)

	nonce, _ := base64.StdEncoding.DecodeString(cr.Metadata.SecureKey.Iv)
	dect := cryptoTypes.DoubleEncryptedCipherText{
		CipherText:     cipherText,
		CipherTextKeys: [][]byte{legalEntityKey},
		Nonce:          nonce,
	}
	consentRecord, err := c.NutsCrypto.DecryptKeyAndCipherText(dect, cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: legalEntity}))
	if err != nil {
		c.logger().WithError(err).Error("Could not decrypt consent record")
		return "", err
	}

	return string(consentRecord), nil
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

func encryptConsentFact(consentFact ConsentFact) (cryptoTypes.DoubleEncryptedCipherText, error) {
	// list of PEM encoded pubic keys to encrypt the record
	var partyKeys []jwk.Key

	// get public key for actor
	registryClient := client.NewRegistryClient()
	organization, err := registryClient.OrganizationById(consentFact.Actor())
	if err != nil {
		logger.Logger().Errorf("error while getting public key for actor: %v from registry: %v\n", consentFact.Actor(), err)
		return cryptoTypes.DoubleEncryptedCipherText{}, err
	}

	jwk, err := organization.CurrentPublicKey()
	if err != nil {
		return cryptoTypes.DoubleEncryptedCipherText{}, fmt.Errorf("registry entry for organization %v does not contain a public key", consentFact.Actor())
	}

	partyKeys = append(partyKeys, jwk)

	// get public key for custodian
	cryptoClient := pkg.NewCryptoClient()
	keyIdentifier := cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: consentFact.Custodian()})
	jwk, err = cryptoClient.GetPublicKeyAsJWK(keyIdentifier)
	if err != nil {
		logger.Logger().Errorf("error while getting public key for custodian: %v from crypto: %v\n", consentFact.Custodian(), err)
		return cryptoTypes.DoubleEncryptedCipherText{}, err
	}
	partyKeys = append(partyKeys, jwk)

	return cryptoClient.EncryptKeyAndPlainText(consentFact.Payload(), partyKeys)
}

// The node can manage more than one legalEntity. This method provides a deterministic way of selecting the current
// legalEntity to work with. It loops over all legalEntities, selects the ones that still needs to sign and selects
// the first one which is managed by this node.
func (c CordaChannel) findFirstEntityToSignFor(signatures *[]bridgeClient.PartyAttachmentSignature, identifiers []bridgeClient.Identifier) string {
	// fill map with signatures legalEntity for easy lookup
	attSignatures := make(map[string]bool)
	// signatures can be nil if no signatures have been set yet
	if signatures != nil {
		for _, att := range *signatures {
			attSignatures[string(att.LegalEntity)] = true
		}

	}

	// Find all LegalEntities managed by this node which still need a signature
	// for each legal entity...
	for _, ent := range identifiers {
		// ... check if it has already signed the request
		if !attSignatures[string(ent)] {
			// if not, check if this node has any keys
			if c.NutsCrypto.PrivateKeyExists(cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: string(ent)})) {
				// yes, so lets add it to the missingSignatures so we can sign it in the next step
				c.logger().Debugf("found first entity to sign for: %v", ent)
				return string(ent)
			}
		}
	}
	return ""
}


// HandleEventConsentRequestValid republishes every event as acked.
// TODO: This should be made optional so the ECD can perform checks and publish the ack or nack
func (c CordaChannel) HandleEventConsentRequestValid(event *events.Event) {
	event, _ = c.autoAckConsentRequest(*event)
	_ = c.Publish(events.ChannelConsentRequest, event)
}

func (c CordaChannel) autoAckConsentRequest(event events.Event) (*events.Event, error) {
	newEvent := event
	newEvent.Name = events.EventConsentRequestAcked
	return &newEvent, nil
}

// HandleEventConsentRequestAcked handles the Event Consent Request Acked event. It passes a copy of the event to the
// signing step and if everything is ok, it publishes this new event to ChannelConsentRequest.
// In case of an error, it publishes the event to ChannelConsentErrored.
func (c CordaChannel) HandleEventConsentRequestAcked(event *events.Event) {
	var newEvent *events.Event
	var err error

	if newEvent, err = c.signConsentRequest(*event); err != nil {
		errorMsg := fmt.Sprintf("%s: could not sign request %v", identity(), err)
		event.Name = events.EventErrored
		event.Error = &errorMsg
		_ = c.Publish(events.ChannelConsentRequest, event)
	}
	newEvent.Name = events.EventAttachmentSigned
	_ = c.Publish(events.ChannelConsentRequest, newEvent)
}

func (c CordaChannel) signConsentRequest(event events.Event) (*events.Event, error) {
	crs := bridgeClient.FullConsentRequestState{}
	decodedPayload, err := base64.StdEncoding.DecodeString(event.Payload)
	if err != nil {
		errorDescription := "Could not base64 decode event payload"
		event.Error = &errorDescription
		c.logger().WithError(err).Error(errorDescription)
		return &event, nil
	}
	if err := json.Unmarshal(decodedPayload, &crs); err != nil {
		// have event-octopus handle redelivery or cancellation
		errorDescription := "Could not unmarshall event payload"
		event.Error = &errorDescription
		c.logger().WithError(err).Error(errorDescription)
		return &event, nil
	}

	for _, cr := range crs.ConsentRecords {
		legalEntityToSignFor := c.findFirstEntityToSignFor(cr.Signatures, crs.LegalEntities)

		// is there work for the given record, otherwise continue till a missing signature is detected
		if legalEntityToSignFor == "" {
			// nothing to sign for this node/record.
			continue
		}

		consentRecordHash := *cr.AttachmentHash
		c.logger().Debugf("signing for LegalEntity %s and consentRecordHash %s", legalEntityToSignFor, consentRecordHash)

		pubKey, err := c.NutsCrypto.GetPublicKeyAsJWK(cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: legalEntityToSignFor}))
		if err != nil {
			c.logger().Errorf("Error in getting pubKey for %s: %v", legalEntityToSignFor, err)
			return nil, err
		}

		jwk, err := cert.JwkToMap(pubKey)
		if err != nil {
			c.logger().Errorf("Error in transforming pubKey for %s: %v", legalEntityToSignFor, err)
			return nil, err
		}
		hexConsentRecordHash, err := hex.DecodeString(consentRecordHash)
		if err != nil {
			c.logger().Errorf("Could not decode consentRecordHash into hex value %s: %v", consentRecordHash, err)
			return nil, err
		}
		sigBytes, err := c.NutsCrypto.Sign(hexConsentRecordHash, cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: legalEntityToSignFor}))
		if err != nil {
			errorDescription := fmt.Sprintf("Could not sign consent record for %s, err: %v", legalEntityToSignFor, err)
			event.Error = &errorDescription
			c.logger().WithError(err).Error(errorDescription)
			return &event, err
		}
		encodedSignatureBytes := base64.StdEncoding.EncodeToString(sigBytes)
		partySignature := bridgeClient.PartyAttachmentSignature{
			Attachment:  consentRecordHash,
			LegalEntity: bridgeClient.Identifier(legalEntityToSignFor),
			Signature: bridgeClient.SignatureWithKey{
				Data:      encodedSignatureBytes,
				PublicKey: bridgeClient.JWK{AdditionalProperties: jwk},
			},
		}

		payload, err := json.Marshal(partySignature)
		if err != nil {
			return nil, err
		}
		event.Payload = base64.StdEncoding.EncodeToString(payload)
		c.logger().Debugf("Consent request signed for %s", legalEntityToSignFor)

		return &event, nil
	}

	errorDescription := fmt.Sprintf("event with name %s recevied, but nothing to sign for this node", events.EventConsentRequestValid)
	event.Error = &errorDescription
	c.logger().WithError(err).Error(errorDescription)
	return &event, err
}

// intermediate struct to keep FHIR resource and hash together
type fhirResourceWithHash struct {
	FHIRResource string
	// Hash represents the attachment hash (zip of cipherText and metadata) from the distributed event model
	Hash string
	// PreviousHash represents the previous attachment hash from the distributed event model (in the case of updates)
	PreviousHash *string
}

func (c CordaChannel) HandleEventConsentDistributed(event *events.Event) {
	c.logger().Debugf("consent request distribyted: %+v\n", event)
	crs := bridgeClient.ConsentState{}
	decodedPayload, err := base64.StdEncoding.DecodeString(event.Payload)
	if err != nil {
		c.logger().Errorf("Unable to base64 decode event payload")
		return
	}
	if err := json.Unmarshal(decodedPayload, &crs); err != nil {
		c.logger().Errorf("Unable to unmarshal event payload")
		return
	}

	var fhirConsents = map[string]fhirResourceWithHash{}

	for _, cr := range crs.ConsentRecords {
		for _, organisation := range cr.Metadata.OrganisationSecureKeys {
			if !c.NutsCrypto.PrivateKeyExists(cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: string(organisation.LegalEntity)})) {
				// this organisation is not managed by this node, try with next
				continue
			}

			fhirConsentString, err := c.decryptConsentRecord(cr, string(organisation.LegalEntity))
			if err != nil {
				c.logger().Error("Could not decrypt fhir consent")
				return
			}
			fhirConsents[*cr.AttachmentHash] = fhirResourceWithHash{
				Hash:         *cr.AttachmentHash,
				PreviousHash: cr.Metadata.PreviousAttachmentHash,
				FHIRResource: fhirConsentString,
			}
		}
	}

	patientConsent := c.PatientConsentFromFHIRRecord(fhirConsents)
	patientConsent.ID = *crs.ConsentId.ExternalId

	if relevant := c.isRelevantForThisNode(patientConsent); !relevant {
		c.logger().Error("Got a patientConsent irrelevant for this node")
		return
	}

	c.logger().Debugf("received patientConsent with %d consentRecords", len(patientConsent.Records))
	c.logger().Debugf("Storing consent: %+v", patientConsent)

	//err = c.NutsConsentStore.RecordConsent(context.Background(), []cStore.PatientConsent{patientConsent})
	//if err != nil {
	//	logger().WithError(err).Error("unable to record the consents")
	//	return
	//}

	event.Name = events.EventCompleted
	err = c.Publisher.Publish(events.ChannelConsentRequest, *event)
	if err != nil {
		c.logger().WithError(err).Error("unable to publish the EventCompleted event")
		return
	}

}
// PatientConsentFromFHIRRecord extracts the PatientConsent from a FHIR consent record encoded as json string.
func (CordaChannel) PatientConsentFromFHIRRecord(fhirConsents map[string]fhirResourceWithHash) cStore.PatientConsent{
	var patientConsent cStore.PatientConsent

	// FixMe: we should add a check if the actors, subjects and custodians are all the same for each of these fhirConsents
	for _, consent := range fhirConsents {
		fhirConsent := gojsonq.New().JSONString(consent.FHIRResource)
		patientConsent.Actor = string(fhirValidator.ActorsFrom(fhirConsent)[0])
		patientConsent.Custodian = fhirValidator.CustodianFrom(fhirConsent)
		patientConsent.Subject = fhirValidator.SubjectFrom(fhirConsent)
		dataClasses := cStore.DataClassesFromStrings(fhirValidator.ResourcesFrom(fhirConsent))
		period := fhirValidator.PeriodFrom(fhirConsent)
		patientConsent.Records = append(patientConsent.Records, cStore.ConsentRecord{DataClasses: dataClasses, ValidFrom: *period[0], ValidTo: period[1], Hash: consent.Hash, PreviousHash: consent.PreviousHash})
	}

	return patientConsent
}

// only consent records of which or the custodian or the actor is managed by this node should be stored
func (c CordaChannel) isRelevantForThisNode(patientConsent cStore.PatientConsent) bool {
	// add if custodian is managed by this node
	return c.NutsCrypto.PrivateKeyExists(cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: patientConsent.Custodian})) ||
		c.NutsCrypto.PrivateKeyExists(cryptoTypes.KeyForEntity(cryptoTypes.LegalEntity{URI: patientConsent.Actor}))
}
