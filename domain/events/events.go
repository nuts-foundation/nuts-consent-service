package events

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"time"
)

const ConsentRequestRegistered = eh.EventType("consent:request-registered")
const ConsentRequestFailed = eh.EventType("consent:request-failed")
const ReservationAccepted = eh.EventType("treatment-relation:consent-reservation-accepted")
const ReservationRejected = eh.EventType("treatment-relation:consent-reservation-rejected")
const NegotiationPrepared = eh.EventType("negotiation:prepared")
const ConsentProposed = eh.EventType("negotiation:proposed")
const ConsentFactGenerated = eh.EventType("negotiation:consent-fact-generated")
const SignatureAdded = eh.EventType("negotiation:signature-added")
const AllSignaturesPresent = eh.EventType("negotiation:all-signatures-present")
const NegotiationStateUpdated = eh.EventType("negotiation:state-updated")

type ConsentData struct {
	ID          uuid.UUID
	CustodianID string
	SubjectID   string
	ActorID     string
	Class       string
	Start       time.Time
	End         time.Time
}

type ConsentFactData struct {
	ConsentID   uuid.UUID
	ConsentFact []byte
}

type NegotiationData struct {
	ConsentFact []byte
}

type SyncStartedData struct {
	SyncID uuid.UUID
}

type FailedData struct {
	Reason string
}

type SignatureData struct {
	SigningParty string
	ConsentID    string
	Signature    string
}

type ChannelStateData struct {
	State interface{}
}

func init() {
	eh.RegisterEventData(ConsentRequestRegistered, func() eh.EventData {
		return &ConsentData{}
	})
	eh.RegisterEventData(ReservationAccepted, func() eh.EventData {
		return &ConsentData{}
	})
	eh.RegisterEventData(NegotiationPrepared, func() eh.EventData {
		return &NegotiationData{}
	})
	eh.RegisterEventData(ConsentFactGenerated, func() eh.EventData {
		return &ConsentFactData{}
	})
	eh.RegisterEventData(SignatureAdded, func() eh.EventData {
		return &SignatureData{}
	})
	eh.RegisterEventData(NegotiationStateUpdated, func() eh.EventData {
		return &ChannelStateData{}
	})
}
