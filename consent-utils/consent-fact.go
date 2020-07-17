package consent_utils

import "github.com/nuts-foundation/nuts-consent-service/domain/events"

// ConsentFactBuilder defines the interface a factbuilder for consents should implement.
// A fact contains information about a process. In this case the consent given to a
// custodian to exchange data with another actor.
// This facts can be exchanged between the parties.
// A fact can be represented in a known domain format like multiple versions of FHIR, or just JSON or a binary format.
type ConsentFactBuilder interface {
	BuildFact(consent events.ConsentData) ([]byte, error)
	VerifyFact([]byte) (bool, error)
}
