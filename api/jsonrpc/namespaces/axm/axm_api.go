package axm

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom/internal/coreapi/api"
	"github.com/axiomesh/axiom/pkg/repo"
)

type AxmAPI struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *repo.Config
	api    api.CoreAPI
	logger logrus.FieldLogger
}

func NewAxmAPI(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) *AxmAPI {
	ctx, cancel := context.WithCancel(context.Background())
	return &AxmAPI{ctx: ctx, cancel: cancel, config: config, api: api, logger: logger}
}

func (api *AxmAPI) Status() any {
	syncStatus := make(map[string]string)
	err := api.api.Broker().OrderReady()
	if err != nil {
		syncStatus["status"] = "abnormal"
		syncStatus["error_msg"] = err.Error()
		return syncStatus
	}
	syncStatus["status"] = "normal"
	return syncStatus
}

func (api *AxmAPI) GetTotalPendingTxCount() any {
	return api.api.Broker().GetTotalPendingTxCount()
}
