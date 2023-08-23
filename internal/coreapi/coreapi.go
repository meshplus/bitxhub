package coreapi

import (
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom/internal/app"
	"github.com/axiomesh/axiom/internal/coreapi/api"
	"github.com/axiomesh/axiom/pkg/loggers"
)

var _ api.CoreAPI = (*CoreAPI)(nil)

type CoreAPI struct {
	axiom  *app.Axiom
	logger logrus.FieldLogger
}

func New(axiom *app.Axiom) (*CoreAPI, error) {
	return &CoreAPI{
		axiom:  axiom,
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

func (api *CoreAPI) Gas() api.GasAPI {
	return (*GasAPI)(api)
}
