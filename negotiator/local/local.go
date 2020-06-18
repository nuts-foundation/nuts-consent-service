package local

import (
	"github.com/google/uuid"
	"log"
)

type LocalNegotiator struct {
}

func (l LocalNegotiator) Start(parties []string, contents string) (uuid.UUID, error) {
	id := uuid.New()
	log.Printf("sync started with id: %s\n", id)
	return id, nil
}
