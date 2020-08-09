package consent_utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)


func TestConsentFactMock(t *testing.T) {
	t.Run("ok - builds one", func(t *testing.T) {
		consentFact := MockConsentFact{
			id:        "1123",
			subject:   "999",
			actor:     "123",
			custodian: "456",
			start:     time.Now(),
			end:       time.Time{},
			hash:      "hash123",
		}
		payload := consentFact.Payload()
		builder := MockConsentFactBuilder{}
		factFromBuilder, err := builder.FactFromBytes(payload)
		assert.NoError(t, err)
		assert.Equal(t, consentFact.subject, factFromBuilder.Subject())
	})
}
