package consent_utils

import (
	"github.com/nuts-foundation/nuts-consent-service/domain/events"
	"time"
)

type ConsentFactBuilder interface {
	BuildFact(consent events.ConsentData) ([]byte, error)
	VerifyFact([]byte) (bool, error)
	FactFromBytes([]byte) (ConsentFact, error)
}

type ConsentFact interface {
	ID() string
	Subject() string
	Actor() string
	Custodian() string
	Start() time.Time
	End() time.Time
	Hash() string
	Payload() []byte
}

