package local

import (
	"github.com/google/uuid"
	"github.com/nuts-foundation/nuts-consent-service/pkg/logger"
)

type LocalNegotiator struct {
}

func (l LocalNegotiator) Start(parties []string, contents string) (uuid.UUID, error) {
	id := uuid.New()
	logger.Logger().Debugf("sync started with id: %s\n", id)
	return id, nil
}
