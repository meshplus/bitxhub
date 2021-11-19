package coreapi

import (
	"github.com/meshplus/bitxhub/internal/app"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/sirupsen/logrus"
)

var _ api.CoreAPI = (*CoreAPI)(nil)

type CoreAPI struct {
	bxh    *app.BitXHub
	logger logrus.FieldLogger
}

func New(bxh *app.BitXHub) (*CoreAPI, error) {
	return &CoreAPI{
		bxh:    bxh,
		logger: loggers.Logger(loggers.CoreAPI),
	}, nil
}

func (api *CoreAPI) Account() api.AccountAPI {
	return (*AccountAPI)(api)
}

func (api *CoreAPI) Broker() api.BrokerAPI {
	return (*BrokerAPI)(api)
}

func (api *CoreAPI) Network() api.NetworkAPI {
	return (*NetworkAPI)(api)
}

func (api *CoreAPI) Chain() api.ChainAPI {
	return (*ChainAPI)(api)
}

func (api *CoreAPI) Feed() api.FeedAPI {
	return (*FeedAPI)(api)
}

func (api *CoreAPI) Audit() api.AuditAPI {
	return (*AuditAPI)(api)
}
