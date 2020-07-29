package consent_utils

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/nuts-foundation/consent-bridge-go-client/api"
	"github.com/nuts-foundation/nuts-crypto/pkg/cert"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
	cryptoClientMock "github.com/nuts-foundation/nuts-crypto/test/mock"
	octopusClientMock "github.com/nuts-foundation/nuts-event-octopus/mock"
	"github.com/nuts-foundation/nuts-event-octopus/pkg"
	registryClientMock "github.com/nuts-foundation/nuts-registry/mock"
	"github.com/nuts-foundation/nuts-registry/pkg/db"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestCordaChannel_ReceiveEvent(t *testing.T) {

	consentRequestState := api.FullConsentRequestState{}
	encodedState, _ := json.Marshal(consentRequestState)

	t.Run("error - sending an event without payload", func(t *testing.T) {
		payload := base64.StdEncoding.EncodeToString([]byte(""))
		event := &(pkg.Event{Name: pkg.EventDistributedConsentRequestReceived, Payload: payload})

		cc := CordaChannel{}
		err := cc.ReceiveEvent(event)
		if assert.Error(t, err) {
			assert.EqualError(t, err, "could not unmarshall cordaEvent payload: unexpected end of JSON input")
		}
	})

	t.Run("testing public keys", func(t *testing.T) {
		const validPublicKey = `{
    "kty": "RSA",
    "n": "uKjoosQFSAYCS-QQGVBh8N-GFd34ufUAdGBwLvvMzB0JPpGpEX0oo8RS4dL8JCruHlzT4HP_bPzIF41fc4WTiOFPFpktY1tJdBS2_XS8i2ehzFLw3YJ3qWX9XQGdJfNHdbbz9h1RXIgBs7UdipHD0-hW-XesT_YkhJSrOA5UxglojI2LrArCzbwlbUUhidMH7962uC87IYvhOux8DK54aOEteNER-ZkZRpnR5vBYT03Soje8KBNez2x-GUlhRDQwS_11PDditMGObAScaJVHrZm-HohiH_rRcQFl0QWLWCFwpPdfu5eHEputNl9GOjvPpRezuvDYN641jL7uZ_rokQ",
    "e": "AQAB"
}`

		validJwk := &api.JWK{}
		_ = json.Unmarshal([]byte(validPublicKey), validJwk)

		signatures := []api.PartyAttachmentSignature{
			{
				LegalEntity: "urn:agb:00000002",
				Signature:   api.SignatureWithKey{Data: "signature", PublicKey: *validJwk},
			},
		}
		consentRequestState.LegalEntities = []api.Identifier{"urn:agb:00000002"}
		consentRequestState.ConsentRecords = []api.ConsentRecord{
			{
				Signatures: &signatures,
			},
		}

		t.Run("ok - organization has multiple valid keys", func(t *testing.T) {
			otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			publisherMock := octopusClientMock.NewMockIEventPublisher(ctrl)
			registryMock := registryClientMock.NewMockRegistryClient(ctrl)
			registryMock.EXPECT().OrganizationById(gomock.Eq("urn:agb:00000002")).Return(getOrganization(&otherKey.PublicKey, validPublicKey), nil)

			encodedState, _ = json.Marshal(consentRequestState)
			payload := base64.StdEncoding.EncodeToString(encodedState)
			event := &(pkg.Event{Name: pkg.EventDistributedConsentRequestReceived, Payload: payload, InitiatorLegalEntity: "urn:agb:00000001"})
			publisherMock.EXPECT().Publish(gomock.Eq(pkg.ChannelConsentRequest), gomock.Any())

			cc := CordaChannel{Publisher: publisherMock, Registry: registryMock}
			err := cc.ReceiveEvent(event)
			assert.NoError(t, err)
		})

		t.Run("error - invalid registry key", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			registryMock := registryClientMock.NewMockRegistryClient(ctrl)
			registryMock.EXPECT().OrganizationById(gomock.Eq("urn:agb:00000002")).Return(&db.Organization{Keys: []interface{}{map[string]interface{}{}}}, nil)

			encodedState, _ = json.Marshal(consentRequestState)
			payload := base64.StdEncoding.EncodeToString(encodedState)
			event := &(pkg.Event{Name: pkg.EventDistributedConsentRequestReceived, Payload: payload, InitiatorLegalEntity: "urn:agb:00000001"})

			cc := CordaChannel{Registry: registryMock}
			err := cc.ReceiveEvent(event)
			if assert.Error(t, err) {
				assert.EqualError(t, err, "could not check JWK against organization keys: failed to construct key from map: unsupported kty type <nil>")
			}
		})

		t.Run("error - invalid signature key", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			registryMock := registryClientMock.NewMockRegistryClient(ctrl)
			registryMock.EXPECT().OrganizationById(gomock.Eq("urn:agb:00000002")).Return(getOrganization(validPublicKey), nil)
			signatures[0].Signature = api.SignatureWithKey{Data: "signature", PublicKey: api.JWK{}}

			encodedState, _ = json.Marshal(consentRequestState)
			payload := base64.StdEncoding.EncodeToString(encodedState)
			event := &(pkg.Event{Name: pkg.EventDistributedConsentRequestReceived, Payload: payload, InitiatorLegalEntity: "urn:agb:00000001"})

			cc := CordaChannel{Registry: registryMock}
			err := cc.ReceiveEvent(event)
			if assert.Error(t, err) {
				assert.EqualError(t, err, "unable to parse signature public key as JWK: failed to construct key from map: unsupported kty type <nil>")
			}
		})

		t.Run("error - org does not have valid cert for used key", func(t *testing.T) {
			withTime(time.Now(), func() {
				otherValidPublicKey := `{
    "kty": "RSA",
    "n": "uKjoosQFSAYCS-QQGVBh8N-GFd34ufUAdGBwLvvMzB0JPpGpEX0oo8RS4dL8JCruHlzT4HP_bPzIF41fc4WTiOFPFpktY1tJdBS2_XS8i2ehzFLw3YJ3qWX9XQGdJfNHdbbz9h1RXIgBs7UdipHD0-hW-XesT_YkhJSrOA5UxglojI2LgArCzbwlbUUhidMH7962uC87IYvhOux8DK54aOEteNER-ZkZRpnR5vBYT03Soje8KBNez2x-GUlhRDQwS_11PDditMGObAScaJVHrZm-HohiH_rRcQFl0QWLWCFwpPdfu5eHEputNl9GOjvPpRezuvDYN641jL7uZ_rokQ",
    "e": "AQAB"
}`
				otherValidJwk := &api.JWK{}
				_ = json.Unmarshal([]byte(otherValidPublicKey), otherValidJwk)
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				registryMock := registryClientMock.NewMockRegistryClient(ctrl)
				registryMock.EXPECT().OrganizationById(gomock.Eq("urn:agb:00000002")).Return(getOrganization(validPublicKey), nil)
				signatures[0].Signature = api.SignatureWithKey{Data: "signature", PublicKey: *otherValidJwk}

				encodedState, _ = json.Marshal(consentRequestState)
				payload := base64.StdEncoding.EncodeToString(encodedState)
				event := &(pkg.Event{Name: pkg.EventDistributedConsentRequestReceived, Payload: payload, InitiatorLegalEntity: "urn:agb:00000001"})

				cc := CordaChannel{Registry: registryMock}
				err := cc.ReceiveEvent(event)
				if assert.Error(t, err) {
					assert.EqualError(t, err, "organization 'urn:agb:00000002' did not have a (valid) corresponding certificate for the public key used to sign the consent")
				}

			})

		})
	})

	t.Run("ok - finalizes when all attachments are signed and initiatorLegalEntity is set", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		publisherMock := octopusClientMock.NewMockIEventPublisher(ctrl)
		registryMock := registryClientMock.NewMockRegistryClient(ctrl)
		publicKey1 := `{
    "kty": "RSA",
    "n": "uKjoosQFSAYCS-QQGVBh8N-GFd34ufUAdGBwLvvMzB0JPpGpEX0oo8RS4dL8JCruHlzT4HP_bPzIF41fc4WTiOFPFpktY1tJdBS2_XS8i2ehzFLw3YJ3qWX9XQGdJfNHdbbz9h1RXIgBs7UdipHD0-hW-XesT_YkhJSrOA5UxglojI2LrArCzbwlbUUhidMH7962uC87IYvhOux8DK54aOEteNER-ZkZRpnR5vBYT03Soje8KBNez2x-GUlhRDQwS_11PDditMGObAScaJVHrZm-HohiH_rRcQFl0QWLWCFwpPdfu5eHEputNl9GOjvPpRezuvDYN641jL7uZ_rokQ",
    "e": "AQAB"
}`

		apiJWK := api.JWK{}
		_ = json.Unmarshal([]byte(publicKey1), &apiJWK)
		registryMock.EXPECT().OrganizationById(gomock.Eq("urn:agb:00000002")).Return(getOrganization(apiJWK.AdditionalProperties), nil)

		cypherText := "foo"
		attachmentHash := "123hash"
		signatures := []api.PartyAttachmentSignature{
			{
				Attachment:  "123",
				LegalEntity: "urn:agb:00000002",
				Signature:   api.SignatureWithKey{Data: "signature", PublicKey: apiJWK},
			},
		}
		consentRequestState.LegalEntities = []api.Identifier{"urn:agb:00000002"}
		consentRequestState.ConsentRecords = []api.ConsentRecord{
			{
				AttachmentHash: &attachmentHash,
				CipherText:     &cypherText,
				Signatures:     &signatures,
			},
		}
		encodedState, _ = json.Marshal(consentRequestState)
		payload := base64.StdEncoding.EncodeToString(encodedState)
		publisherMock.EXPECT().Publish(gomock.Eq(pkg.ChannelConsentRequest), pkg.Event{Name: pkg.EventAllSignaturesPresent, Payload: payload, InitiatorLegalEntity: "urn:agb:00000001"})

		event := &(pkg.Event{Name: pkg.EventDistributedConsentRequestReceived, Payload: payload, InitiatorLegalEntity: "urn:agb:00000001"})

		cc := CordaChannel{Publisher: publisherMock, Registry: registryMock}
		err := cc.ReceiveEvent(event)
		assert.NoError(t, err)
	})

	t.Run("ok - when no signatures needed and this node is not the initiator it returns without events", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		publisherMock := octopusClientMock.NewMockIEventPublisher(ctrl)

		payload := base64.StdEncoding.EncodeToString(encodedState)
		event := &(pkg.Event{Name: pkg.EventDistributedConsentRequestReceived, Payload: payload})

		cc := CordaChannel{Publisher: publisherMock}
		err := cc.ReceiveEvent(event)
		assert.NoError(t, err)
	})

	t.Run("ok - no signatures set, but remaining LegalEntity not managed by this node should return", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		publisherMock := octopusClientMock.NewMockIEventPublisher(ctrl)
		cryptoMock := cryptoClientMock.NewMockClient(ctrl)

		cryptoMock.EXPECT().PrivateKeyExists(types.KeyForEntity(types.LegalEntity{URI: "urn:agb:00000001"}))
		consentRequestState.LegalEntities = []api.Identifier{"urn:agb:00000001"}
		foo := "foo"
		consentRequestState.ConsentRecords = []api.ConsentRecord{
			{
				CipherText: &foo,
			},
		}
		encodedState, _ := json.Marshal(consentRequestState)
		payload := base64.StdEncoding.EncodeToString(encodedState)

		event := &(pkg.Event{
			Name:    pkg.EventDistributedConsentRequestReceived,
			Payload: payload,
		})

		cc := CordaChannel{Publisher: publisherMock, NutsCrypto: cryptoMock}
		err := cc.ReceiveEvent(event)
		assert.NoError(t, err)
	})

	t.Run("ok - not all signatures set, but remaining LegalEntity not managed by this node should return", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		publisherMock := octopusClientMock.NewMockIEventPublisher(ctrl)
		cryptoMock := cryptoClientMock.NewMockClient(ctrl)
		cryptoMock.EXPECT().PrivateKeyExists(types.KeyForEntity(types.LegalEntity{URI: "urn:agb:00000002"}))
		foo := "foo"
		signatures := []api.PartyAttachmentSignature{{Attachment: "foo", LegalEntity: "urn:agb:00000001"}}
		consentRequestState.ConsentRecords = []api.ConsentRecord{
			{
				CipherText: &foo,
				Signatures: &signatures,
			},
		}
		consentRequestState.LegalEntities = []api.Identifier{"urn:agb:00000001", "urn:agb:00000002"}

		encodedState, _ := json.Marshal(consentRequestState)
		payload := base64.StdEncoding.EncodeToString(encodedState)

		event := &(pkg.Event{
			Name:    pkg.EventDistributedConsentRequestReceived,
			Payload: payload,
		})

		cc := CordaChannel{Publisher: publisherMock, NutsCrypto: cryptoMock}
		err := cc.ReceiveEvent(event)
		assert.NoError(t, err)
	})

	t.Run("ok - not all signatures set, and remaining LegalEntity managed by this node and valid content should broadcast all checks passed", func(t *testing.T) {
		fooEncoded := base64.StdEncoding.EncodeToString([]byte("foo"))

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		publisherMock := octopusClientMock.NewMockIEventPublisher(ctrl)
		cryptoMock := cryptoClientMock.NewMockClient(ctrl)

		cypherText2 := "cyphertext for 00000002"
		// 00000001 already signed
		signatures := []api.PartyAttachmentSignature{{Attachment: "foo", LegalEntity: "urn:agb:00000001"}}
		consentRequestState.ConsentRecords = []api.ConsentRecord{
			{
				CipherText: &fooEncoded,
				Metadata: &api.Metadata{
					OrganisationSecureKeys: []api.ASymmetricKey{{LegalEntity: "urn:agb:00000002", CipherText: &cypherText2}},
				},
				Signatures: &signatures,
			},
		}
		// two parties involved in this transaction
		consentRequestState.LegalEntities = []api.Identifier{"urn:agb:00000001", "urn:agb:00000002"}
		// 00000002 is managed by this node
		cryptoMock.EXPECT().PrivateKeyExists(types.KeyForEntity(types.LegalEntity{URI: "urn:agb:00000002"})).Return(true)

		// expect to receive a decrypt call for 00000002
		validConsent, err := ioutil.ReadFile("../test-data/valid-consent.json")
		if err != nil {
			t.Error(err)
		}
		cryptoMock.EXPECT().DecryptKeyAndCipherText(gomock.Any(), types.KeyForEntity(types.LegalEntity{URI: "urn:agb:00000002"})).Return(validConsent, nil)

		encodedState, _ := json.Marshal(consentRequestState)
		payload := base64.StdEncoding.EncodeToString(encodedState)
		event := &(pkg.Event{
			Name:    pkg.EventDistributedConsentRequestReceived,
			Payload: payload,
		})

		expectedEvent := event
		expectedEvent.Name = pkg.EventConsentRequestValid

		// expect to receive a all check passed event
		publisherMock.EXPECT().Publish(gomock.Eq(pkg.ChannelConsentRequest), *expectedEvent)

		cc := CordaChannel{Publisher: publisherMock, NutsCrypto: cryptoMock}
		err = cc.ReceiveEvent(event)
		assert.NoError(t, err)
	})

}

// getOrganization helper func to create organization with the given (mixed-format) keys. The keys can be in the following formats:
// - PEM encoded public key as string
// - JSON encoded JWK as string
// - JWK as Go map[string]interface{}
// - RSA public key as Go *rsa.PublicKey
func getOrganization(keys ...interface{}) *db.Organization {
	o := db.Organization{}
	for _, key := range keys {
		var keyAsJWK jwk.Key
		var err error
		{
			keyAsString, ok := key.(string)
			if ok {
				keyAsJWK, _ = cert.PemToJwk([]byte(keyAsString))
				if keyAsJWK == nil {
					var asMap map[string]interface{}
					err := json.Unmarshal([]byte(keyAsString), &asMap)
					if err == nil {
						key = asMap
					}
				}
			}
		}
		{
			keyAsMap2, ok := key.(map[string]interface{})
			if ok {
				keyAsJWK, err = cert.MapToJwk(keyAsMap2)
				if err != nil {
					panic(err)
				}
			}
		}
		{
			keyAsPubKey, ok := key.(*rsa.PublicKey)
			if ok {
				keyAsJWK, _ = jwk.New(keyAsPubKey)
			}
		}
		keyAsMap, _ := cert.JwkToMap(keyAsJWK)
		keyAsMap["kty"] = keyAsJWK.KeyType().String()
		o.Keys = append(o.Keys, keyAsMap)
	}
	return &o
}

func withTime(testTime time.Time, testFn func()) {
	timeFn := TimeNow

	TimeNow = func() time.Time {
		return testTime
	}
	defer func() {
		TimeNow = timeFn
	}()

	testFn()
}
