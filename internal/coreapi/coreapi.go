package coreapi

import (
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-ledger/internal/app"
	"github.com/axiomesh/axiom-ledger/internal/coreapi/api"
	"github.com/axiomesh/axiom-ledger/pkg/loggers"
)

var _ api.CoreAPI = (*CoreAPI)(nil)

type CoreAPI struct {
	axiomLedger *app.AxiomLedger
	logger      logrus.FieldLogger
}

func New(axiomLedger *app.AxiomLedger) (*CoreAPI, error) {
	return &CoreAPI{
		axiomLedger: axiomLedger,
		logger:      loggers.Logger(loggers.CoreAPI),
	}, nil
}

func (api *CoreAPI) Broker() api.BrokerAPI {
	return (*BrokerAPI)(api)
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
