package domain

import (
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

const ConsentAggregateType = eh.AggregateType("consent")

const TreatmentRelationAggregateType = eh.AggregateType("treatment-relation")

var NutsExternalIDSpace = uuid.Must(uuid.Parse("6ba7b812-9dad-11d1-80b4-00c04fd430c8"))
