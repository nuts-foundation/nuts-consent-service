package consent_utils

import (
	"crypto/sha256"
	"fmt"
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"github.com/nuts-foundation/nuts-fhir-validation/pkg"
	"github.com/thedevsaddam/gojsonq/v2"
	"time"
)

// ConsentFactBuilder defines the interface a factbuilder for consents should implement.
// A fact contains information about a process. In this case the consent given to a
// custodian to exchange data with another actor.
// This facts can be exchanged between the parties.
// A fact can be represented in a known domain format like multiple versions of FHIR, or just JSON or a binary format.
type ConsentFactBuilder interface {
	BuildFact(consent events.ConsentData) ([]byte, error)
	VerifyFact([]byte) (bool, error)
}

type ConsentFactProperties interface {
	Subject() string
	Actor() string
	Custodian() string
	Start() time.Time
	End() time.Time
	Hash() string
}

type ConsentFact struct {
	Payload []byte
}

func (c ConsentFact) Actor() string {
	jsonq := gojsonq.New().FromString(string(c.Payload))
	identifier := pkg.ActorsFrom(jsonq)[0]
	return string(identifier)
}

func (c ConsentFact) Custodian() string {
	jsonq := gojsonq.New().FromString(string(c.Payload))
	return pkg.CustodianFrom(jsonq)
}

func (c ConsentFact) Start() time.Time {
	jsonq := gojsonq.New().FromString(string(c.Payload))
	return *pkg.PeriodFrom(jsonq)[0]
}

func (c ConsentFact) End() time.Time {
	jsonq := gojsonq.New().FromString(string(c.Payload))
	end := pkg.PeriodFrom(jsonq)[1]
	if end != nil {
		return *end
	}
	return time.Time{}
}

func (c ConsentFact) Subject() string {
	jsonq := gojsonq.New().FromString(string(c.Payload))
	return pkg.SubjectFrom(jsonq)
}

func (c ConsentFact) Hash() string {
	return fmt.Sprintf("%x", sha256.Sum256(c.Payload))
}

