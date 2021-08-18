package profile

import (
	"fmt"
	"net/http"

	"github.com/meshplus/bitxhub/internal/loggers"

	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Monitor struct {
	enable bool
	port   int64
	server *http.Server
	logger logrus.FieldLogger
}

func NewMonitor(config *repo.Config) (*Monitor, error) {
	monitor := &Monitor{
		enable: config.Monitor.Enable,
		port:   config.Port.Monitor,
		logger: loggers.Logger(loggers.Profile),
	}

	monitor.init()

	return monitor, nil
}

func (m *Monitor) init() {
	if m.enable {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		addr := fmt.Sprintf(":%d", m.port)
		m.server = &http.Server{
			Addr:    addr,
			Handler: mux,
		}
	} else {
		m.server = nil
	}
}

// Start start prometheus monitor
func (m *Monitor) Start() error {
	if m.enable {
		m.logger.WithField("port", m.port).Info("Start monitor")
		go func() {
			err := m.server.ListenAndServe()
			if err != nil {
				fmt.Println(err)
			}
		}()
	}

	return nil
}

// Stop start prometheus monitor
func (m *Monitor) Stop() error {
	if m.enable {
		m.logger.WithField("port", m.port).Info("Stop monitor")
		return m.server.Close()
	}

	return nil
}

// ReConfig reconfigure prometheus monitor
func (m *Monitor) ReConfig(config *repo.Config) error {
	if m.enable != config.Monitor.Enable || m.port != config.Port.Monitor {
		if err := m.Stop(); err != nil {
			return err
		}
		m.enable = config.Monitor.Enable
		m.port = config.Port.Monitor

		m.init()

		if err := m.Start(); err != nil {
			return err
		}
	}

	return nil
}
