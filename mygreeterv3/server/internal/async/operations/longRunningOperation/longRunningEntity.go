package longRunningOperation

import (
	"github.com/Azure/aks-async/runtime/entity"
)

// Setting the variable to ensure all functions of the Entity interface are implemented.
var _ entity.Entity = &LongRunningEntity{}

type LongRunningEntity struct {
	LastOperationId string
}

func NewLongRunningEntity(lastOperationId string) *LongRunningEntity {
	return &LongRunningEntity{
		LastOperationId: lastOperationId,
	}
}

func (lre LongRunningEntity) GetLatestOperationID() string {
	return lre.LastOperationId
}
