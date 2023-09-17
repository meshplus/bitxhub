package profile

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom/pkg/loggers"
	"github.com/axiomesh/axiom/pkg/repo"
)

type Pprof struct {
	repoRoot string
	config   *repo.PProf
	port     int64
	logger   logrus.FieldLogger
	server   *http.Server
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewPprof(config *repo.Config) (*Pprof, error) {
	pprof := &Pprof{
		repoRoot: config.RepoRoot,
		config:   &config.PProf,
		port:     config.Port.PProf,
		logger:   loggers.Logger(loggers.Profile),
	}

	pprof.init()

	return pprof, nil
}

func (p *Pprof) init() {
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.server = &http.Server{
		Addr: fmt.Sprintf(":%d", p.port),
	}
}

func (p *Pprof) Start() error {
	if p.config.Enable {
		switch p.config.PType {
		case repo.PprofTypeRuntime:
			go p.runtimePProf()
		case repo.PprofTypeHTTP:
			go p.httpPProf()
		default:
			p.logger.Warnf("unknown ptype: %s", p.config.PType)
		}
	}

	return nil
}

func (p *Pprof) Stop() error {
	if p.config.Enable {
		switch p.config.PType {
		case repo.PprofTypeRuntime:
			p.logger.Info("Stop runtime profile")
			p.cancel()
		case repo.PprofTypeHTTP:
			p.logger.Info("Stop http profile")
			return p.server.Close()
		}
	}

	return nil
}

// runtimePProf will record the cpu or memory profiles every 5 second.
func (p *Pprof) runtimePProf() {
	p.logger.Info("Start runtime pprof")
	tick := time.NewTicker(p.config.Duration.ToDuration())
	rootPath := filepath.Join(p.repoRoot, "/pprof/")
	exist := fileExist(rootPath)
	if !exist {
		err := os.Mkdir(rootPath, os.ModePerm)
		if err != nil {
			fmt.Printf("----- runtimePProf start failed, err: %s -----\n", err.Error())
			return
		}
	}

	var cpuFile *os.File
	if p.config.Mode == repo.PprofModeCpu {
		subPath := fmt.Sprint("cpu-", time.Now().Format("20060102-15:04:05"))
		cpuPath := filepath.Join(rootPath, subPath)
		cpuFile, _ = os.Create(cpuPath)
		_ = pprof.StartCPUProfile(cpuFile)
	}
	for {
		select {
		case <-tick.C:
			switch p.config.Mode {
			case repo.PprofModeCpu:
				pprof.StopCPUProfile()
				_ = cpuFile.Close()
				subPath := fmt.Sprint("cpu-", time.Now().Format("20060102-15:04:05"))
				cpuPath := filepath.Join(rootPath, subPath)
				cpuFile, _ := os.Create(cpuPath)
				_ = pprof.StartCPUProfile(cpuFile)
			case repo.PprofModeMem:
				subPath := fmt.Sprint("mem-", time.Now().Format("20060102-15:04:05"))
				memPath := filepath.Join(rootPath, subPath)
				memFile, _ := os.Create(memPath)
				_ = pprof.WriteHeapProfile(memFile)
				_ = memFile.Close()
			}
		case <-p.ctx.Done():
			if p.config.Mode == repo.PprofModeCpu {
				pprof.StopCPUProfile()
			}
			return
		}
	}
}

func (p *Pprof) httpPProf() {
	p.logger.WithField("port", p.port).Info("Start http pprof")
	err := p.server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}
