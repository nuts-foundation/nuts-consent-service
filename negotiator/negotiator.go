package negotiator

import "github.com/google/uuid"

type Negotiator interface {
	Start(parties []string, contents string) (uuid.UUID, error)
}
