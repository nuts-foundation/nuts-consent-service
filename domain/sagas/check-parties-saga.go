package sagas

import (
	"context"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/eventhandler/saga"
	"github.com/nuts-foundation/nuts-crypto/pkg"
	"github.com/nuts-foundation/nuts-crypto/pkg/types"
)

const CheckPartiesSagaType = saga.Type("CheckPartiesSagaType")

type CheckPartiesSaga struct {
}

func (c CheckPartiesSaga) SagaType() saga.Type {
	return CheckPartiesSagaType
}

func (c CheckPartiesSaga) RunSaga(ctx context.Context, event eh.Event) []eh.Command {
	switch event.EventType() {
	default:

	}
	//case events.Proposed:
	//	data, ok := event.Data().(events.ProposedData)
	//	if !ok {
	//		return []eh.Command{&commands.MarkAsErrored{
	//			ID:     event.AggregateID(),
	//			Reason: "event did not contain proposedData",
	//		}}
	//	}
	//
	//	if c.CheckCustodian(data.CustodianID) {
	//		return []eh.Command{&commands.MarkCustodianChecked{
	//			ID:     event.AggregateID(),
	//		}}
	//	} else {
	//		return []eh.Command{&commands.MarkAsErrored{
	//			ID:     event.AggregateID(),
	//			Reason: "custodian is not a valid or known party",
	//		}}
	//	}
	//
	//}
	return nil
}

func (CheckPartiesSaga) CheckCustodian(custodianID string) bool {
	crypto := pkg.NewCryptoClient()
	legalEntity := types.LegalEntity{URI: custodianID}
	entityKey := types.KeyForEntity(legalEntity)
	return crypto.PrivateKeyExists(entityKey)
}
