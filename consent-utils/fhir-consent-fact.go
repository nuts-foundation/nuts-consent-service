package consent_utils

import (
	"crypto/sha256"
	"fmt"
	nutsFhirValidation "github.com/nuts-foundation/nuts-fhir-validation/pkg"
	"github.com/thedevsaddam/gojsonq/v2"
	"time"
)

// ConsentFactBuilder defines the interface a factbuilder for consents should implement.
// A fact contains information about a process. In this case the consent given to a
// custodian to exchange data with another actor.
// This facts can be exchanged between the parties.
// A fact can be represented in a known domain format like multiple versions of FHIR, or just JSON or a binary format.
type FhirConsentFact struct {
	payload []byte
}

func (c FhirConsentFact) ID() string {
	jsonq := gojsonq.New().FromString(string(c.payload))
	id, ok := jsonq.Copy().Find("id").(string)
	if ok {
		return id
	}
	return ""
}

func (c FhirConsentFact) Actor() string {
	jsonq := gojsonq.New().FromString(string(c.payload))
	identifier := nutsFhirValidation.ActorsFrom(jsonq)[0]
	return string(identifier)
}

func (c FhirConsentFact) Custodian() string {
	jsonq := gojsonq.New().FromString(string(c.payload))
	return nutsFhirValidation.CustodianFrom(jsonq)
}

func (c FhirConsentFact) Start() time.Time {
	jsonq := gojsonq.New().FromString(string(c.payload))
	return *nutsFhirValidation.PeriodFrom(jsonq)[0]
}

func (c FhirConsentFact) End() time.Time {
	jsonq := gojsonq.New().FromString(string(c.payload))
	end := nutsFhirValidation.PeriodFrom(jsonq)[1]
	if end != nil {
		return *end
	}
	return time.Time{}
}

func (c FhirConsentFact) Subject() string {
	jsonq := gojsonq.New().FromString(string(c.payload))
	return nutsFhirValidation.SubjectFrom(jsonq)
}

func (c FhirConsentFact) Hash() string {
	return fmt.Sprintf("%x", sha256.Sum256(c.payload))
}

func (c FhirConsentFact) Payload() []byte {
	return c.payload
}

