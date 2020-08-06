package consent_utils

import (
	"encoding/json"
	events2 "github.com/nuts-foundation/nuts-consent-service/domain/events"
	"time"
)

type MockConsentFactBuilder struct{}

func (m MockConsentFactBuilder) BuildFact(consent events2.ConsentData) ([]byte, error) {
	fact := MockConsentFact{
		id:        consent.ID.String(),
		subject:   consent.SubjectID,
		actor:     consent.ActorID,
		custodian: consent.CustodianID,
		start:     consent.Start,
		end:       consent.End,
		hash:      "hash123",
	}
	return fact.Payload(), nil
}

func (m MockConsentFactBuilder) VerifyFact(bytes []byte) (bool, error) {
	return true, nil
}

func (m MockConsentFactBuilder) FactFromBytes(bytes []byte) (ConsentFact, error) {
	fact := &MockConsentFact{}
	json.Unmarshal(bytes, fact)
	return *fact, nil
}

type MockConsentFact struct {
	id        string
	subject   string
	actor     string
	custodian string
	start     time.Time
	end       time.Time
	hash      string
}

func (m *MockConsentFact) UnmarshalJSON(bytes []byte) error {
	s := &struct {
		Id        string
		Subject   string
		Actor     string
		Custodian string
		Start     time.Time
		End       time.Time
		Hash      string
	}{}
	if err := json.Unmarshal(bytes, s); err != nil {
		return err
	}
	m.id = s.Id
	m.subject = s.Subject
	m.actor = s.Actor
	m.custodian = s.Custodian
	m.start = s.Start
	m.end = s.End
	m.hash = s.Hash

	return nil
}

func (m MockConsentFact) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id        string
		Subject   string
		Actor     string
		Custodian string
		Start     time.Time
		End       time.Time
		Hash      string
	}{
		Id:        m.id,
		Subject:   m.subject,
		Actor:     m.actor,
		Custodian: m.custodian,
		Start:     m.start,
		End:       m.end,
		Hash:      m.hash,
	})
}

func (m MockConsentFact) ID() string {
	return m.id
}

func (m MockConsentFact) Subject() string {
	return m.subject
}

func (m MockConsentFact) Actor() string {
	return m.actor
}

func (m MockConsentFact) Custodian() string {
	return m.custodian
}

func (m MockConsentFact) Start() time.Time {
	return m.start
}

func (m MockConsentFact) End() time.Time {
	return m.end
}

func (m MockConsentFact) Hash() string {
	return m.hash
}

func (m MockConsentFact) Payload() []byte {
	str, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return str
}

