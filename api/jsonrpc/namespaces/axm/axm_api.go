package axm

import (
	"context"

	"github.com/axiomesh/axiom/internal/coreapi/api"
	"github.com/axiomesh/axiom/pkg/repo"
	"github.com/sirupsen/logrus"
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
		syncStatus["result"] = "abnormal"
		syncStatus["orderStatus"] = err.Error()
		return syncStatus
	}
	syncStatus["result"] = "normal"
	return syncStatus
}
